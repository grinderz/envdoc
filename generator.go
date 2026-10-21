package main

import (
	"fmt"
	"io"

	"github.com/g4s8/envdoc/ast"
	"github.com/g4s8/envdoc/debug"
	"github.com/g4s8/envdoc/enums"
	"github.com/g4s8/envdoc/resolver"
	"github.com/g4s8/envdoc/types"
)

type Renderer interface {
	Render(scopes []*types.EnvScope, out io.Writer) error
}

type Generator struct {
	parser    *ast.Parser
	converter *Converter
	renderer  Renderer
}

func NewGenerator(parser *ast.Parser, converter *Converter, renderer Renderer) *Generator {
	return &Generator{
		parser:    parser,
		converter: converter,
		renderer:  renderer,
	}
}

func (g *Generator) Generate(dir string, out io.Writer) error {
	return g.GenerateDirs([]string{dir}, out)
}

// GenerateDirs parses all dirs together, so types from one dir resolve
// fields in another.
func (g *Generator) GenerateDirs(dirs []string, out io.Writer) error {
	var files []*ast.FileSpec
	for _, dir := range dirs {
		dirFiles, err := g.parser.Parse(dir)
		if err != nil {
			return fmt.Errorf("parse dir %q: %w", dir, err)
		}
		files = append(files, dirFiles...)
	}
	enumValues, err := enums.Collect(dirs)
	if err != nil {
		return fmt.Errorf("collect enums: %w", err)
	}
	g.converter.enums = enumValues

	res := resolver.ResolveAllTypes(files)
	debug.PrintDebug(res)

	scopes := g.converter.ScopesFromFiles(res, files)
	printScopesTree(scopes)

	if err := g.renderer.Render(scopes, out); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}
