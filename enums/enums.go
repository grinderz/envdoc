// Package enums collects allowed values of enum-like types: named types with
// a builtin underlying type and a const block, e.g.
//
//	type ModeEnum int
//
//	const (
//		ModeUnknown ModeEnum = iota // unknown
//		ModeFast    ModeEnum = iota // fast
//	)
//
// A const value is its line comment (stringer -linecomment), or its string
// literal for string types. A type is an enum only if every const of it has
// one of those, so typed constants like sizes or timeouts are not listed.
// The `unknown` value and unexported consts of an exported type are skipped:
// they mark an unset value or a bound, not a valid one.
package enums

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

const unknownValue = "unknown"

// Values maps `pkg.Type` (package name, as the resolver uses) to enum values.
type Values map[string][]string

// Get returns values of type name in package pkg.
func (v Values) Get(pkg, name string) []string {
	return v[pkg+"."+name]
}

type constValue struct {
	value string
	ok    bool // value comes from a line comment or a string literal
}

type collector struct {
	types  map[string]bool // named types with a builtin underlying type
	consts map[string][]constValue
}

// Collect parses go files in dirs and their subdirs, test files excluded.
func Collect(dirs []string) (Values, error) {
	c := collector{
		types:  make(map[string]bool),
		consts: make(map[string][]constValue),
	}
	fset := token.NewFileSet()
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				return fmt.Errorf("parse %q: %w", path, err)
			}
			c.addFile(f)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("collect enums in %q: %w", dir, err)
		}
	}
	return c.values(), nil
}

func (c *collector) addFile(f *ast.File) {
	pkg := f.Name.Name
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch gen.Tok {
		case token.TYPE:
			for _, spec := range gen.Specs {
				ts := spec.(*ast.TypeSpec)
				if ident, ok := ts.Type.(*ast.Ident); ok && ts.Assign == 0 && isBuiltin(ident.Name) {
					c.types[pkg+"."+ts.Name.Name] = true
				}
			}
		case token.CONST:
			c.addConsts(pkg, gen)
		}
	}
}

func (c *collector) addConsts(pkg string, gen *ast.GenDecl) {
	// a spec without type and values repeats the previous one (iota blocks)
	var typeName string
	for _, spec := range gen.Specs {
		vs := spec.(*ast.ValueSpec)
		switch {
		case vs.Type != nil:
			typeName = ""
			if ident, ok := vs.Type.(*ast.Ident); ok {
				typeName = ident.Name
			}
		case len(vs.Values) > 0:
			typeName = ""
		}
		if typeName == "" {
			continue
		}
		key := pkg + "." + typeName
		for i, name := range vs.Names {
			// unexported consts of an exported type are sentinels like `typeCount`
			if name.Name == "_" || ast.IsExported(typeName) && !name.IsExported() {
				continue
			}
			c.consts[key] = append(c.consts[key], constSpecValue(vs, i))
		}
	}
}

func constSpecValue(vs *ast.ValueSpec, i int) constValue {
	if comment := strings.TrimSpace(vs.Comment.Text()); comment != "" {
		return constValue{value: comment, ok: true}
	}
	if i < len(vs.Values) {
		if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				return constValue{value: s, ok: true}
			}
		}
	}
	return constValue{value: vs.Names[i].Name}
}

func (c *collector) values() Values {
	res := make(Values)
	for key, consts := range c.consts {
		if !c.types[key] {
			continue
		}
		values := make([]string, 0, len(consts))
		for _, cv := range consts {
			if !cv.ok {
				values = nil
				break
			}
			if !strings.EqualFold(cv.value, unknownValue) {
				values = append(values, cv.value)
			}
		}
		if len(values) > 0 {
			res[key] = values
		}
	}
	return res
}

func isBuiltin(name string) bool {
	switch name {
	case "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}
