package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/g4s8/envdoc/ast"
	"github.com/g4s8/envdoc/render"
	"github.com/g4s8/envdoc/types"
)

// Fork features: cleanenv target with markdown-table output, env-description,
// enum values and map format, types resolved from additional source dirs.
func TestForkCleanenvMarkdownTable(t *testing.T) {
	// relative paths as in go:generate lines: -files=./app/config.go -dir ./app ./lib
	t.Chdir(t.TempDir())
	appDir, libDir := "./app", "./lib"
	for _, dir := range []string{appDir, libDir} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %s", dir, err)
		}
	}
	writeFile(t, filepath.Join(appDir, "config.go"), `package config

import "example.com/libzap"

// Config is the app config.
type Config struct {
	// Log settings.
	Log libzap.Config `+"`env-prefix:\"LOG__\"`"+`
	// Hosts to listen on.
	Hosts []string `+"`env:\"HOSTS\" env-required:\"true\"`"+`
	// Overridden by env-description.
	Mode string `+"`env:\"MODE\" env-default:\"a,b\" env-description:\"Mode, one of a,b.\"`"+`
	// Labels to add.
	Labels map[string]string `+"`env:\"LABELS\"`"+`
}
`)
	writeFile(t, filepath.Join(libDir, "config.go"), `package libzap

// Config is the logger config.
type Config struct {
	// Log level.
	Level string `+"`env:\"LEVEL\" env-default:\"info\"`"+`
	// Output file settings.
	File FileConfig `+"`env-prefix:\"FILE__\"`"+`
	// Log encoding.
	Encoding EncodingEnum `+"`env:\"ENCODING\" env-default:\"json\"`"+`
}

type EncodingEnum int

const (
	EncodingUnknown EncodingEnum = iota // unknown
	EncodingConsole EncodingEnum = iota // console
	EncodingJSON    EncodingEnum = iota // json
)

// FileConfig is the log file config.
type FileConfig struct {
	// Logs dir.
	Dir string `+"`env:\"DIR\"`"+`
}
`)

	p := ast.NewParser("./app/config.go", "*")
	conv := NewConverter(types.TargetTypeCleanenv, ConverterOpts{})
	rend := render.NewRenderer(types.OutFormatMarkdownTable, false, false)
	gen := NewGenerator(p, conv, rend)
	var out bytes.Buffer
	if err := gen.GenerateDirs([]string{libDir, appDir}, &out); err != nil {
		t.Fatalf("generate: %s", err)
	}

	expect := "# Environment Variables\n" +
		"\n" +
		"## Config\n" +
		"\n" +
		"Config is the app config.\n" +
		"\n" +
		"| env variable | type | description | options |\n" +
		"|--------------|------|-------------|---------|\n" +
		"| LOG__LEVEL | `string` | Log level. | (default: `info`) |\n" +
		"| LOG__FILE__DIR | `string` | Logs dir. |  |\n" +
		"| LOG__ENCODING | `EncodingEnum` | Log encoding. | one of: `console`, `json` (default: `json`) |\n" +
		"| HOSTS | `[]string` | Hosts to listen on. | (comma-separated, **required**) |\n" +
		"| MODE | `string` | Mode, one of a,b. | (default: `a,b`) |\n" +
		"| LABELS | `map[string]string` | Labels to add. | `key:value` pairs (comma-separated) |\n" +
		"\n"
	if got := out.String(); got != expect {
		t.Fatalf("unexpected output:\n%s\nexpected:\n%s", got, expect)
	}
}

func writeFile(t *testing.T, name, data string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o600); err != nil {
		t.Fatalf("write %s: %s", name, err)
	}
}

func TestForkCleanenvFieldDecoder(t *testing.T) {
	for _, test := range []struct {
		name         string
		opts         FieldDecoderOpts
		spec         *ast.FieldSpec
		expectField  FieldInfo
		expectPrefix string
	}{
		{
			name: "description",
			spec: &ast.FieldSpec{
				Names:   []string{"Foo"},
				Doc:     "foo doc",
				Tag:     `env:"FOO" env-description:"Foo, bar and baz."`,
				TypeRef: ast.FieldTypeRef{Name: "string", Kind: ast.FieldTypeIdent},
			},
			expectField: FieldInfo{
				Names:       []string{"FOO"},
				Description: "Foo, bar and baz.",
			},
		},
		{
			name: "slice default separator",
			spec: &ast.FieldSpec{
				Names:   []string{"Foo"},
				Tag:     `env:"FOO"`,
				TypeRef: ast.FieldTypeRef{Name: "string", Kind: ast.FieldTypeArray},
			},
			expectField: FieldInfo{
				Names:     []string{"FOO"},
				Separator: ",",
			},
		},
		{
			name: "map default separator",
			spec: &ast.FieldSpec{
				Names:   []string{"Foo"},
				Tag:     `env:"FOO"`,
				TypeRef: ast.FieldTypeRef{Name: "string", Kind: ast.FieldTypeMap},
			},
			expectField: FieldInfo{
				Names:     []string{"FOO"},
				Separator: ",",
			},
		},
		{
			name: "prefix without name",
			opts: FieldDecoderOpts{
				EnvPrefix: "X_",
			},
			spec: &ast.FieldSpec{
				Names:   []string{"Foo"},
				Tag:     `env-prefix:"BAR_"`,
				TypeRef: ast.FieldTypeRef{Name: "Bar", Kind: ast.FieldTypeIdent},
			},
			expectField: FieldInfo{
				Names: []string{""},
			},
			expectPrefix: "X_BAR_",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, prefix := NewFieldDecoder(types.TargetTypeCleanenv, test.opts).Decode(test.spec)
			assertEqFieldInfo(t, test.expectField, res)
			if res.Description != test.expectField.Description {
				t.Errorf("expected description %q, got %q", test.expectField.Description, res.Description)
			}
			if prefix != test.expectPrefix {
				t.Errorf("expected prefix %q, got %q", test.expectPrefix, prefix)
			}
		})
	}
}
