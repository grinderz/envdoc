package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/g4s8/envdoc/types"
)

func TestRendererTemplateFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "custom.tmpl")
	src := `{{- range .Sections }}{{ range .Items }}{{ .EnvName }}: {{ template "item.options" (list . $.Config.Item "(%s)") }}
{{ end }}{{ end -}}`
	if err := os.WriteFile(name, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	r := NewRenderer(types.OutFormatMarkdownTable, false, false)
	if err := r.SetTemplateFile(name); err != nil {
		t.Fatalf("set template: %s", err)
	}
	scopes := []*types.EnvScope{{
		Name: "scope",
		Vars: []*types.EnvDocItem{{
			Name: "FOO",
			Opts: types.EnvVarOptions{Required: true, Default: "bar"},
		}},
	}}
	var sb strings.Builder
	if err := r.Render(scopes, &sb); err != nil {
		t.Fatalf("render: %s", err)
	}
	// option strings come from the -format config
	if expect, got := "FOO: (**required**, default: `bar`)\n", sb.String(); got != expect {
		t.Fatalf("expected %q, got %q", expect, got)
	}

	if err := r.SetTemplateFile(filepath.Join(t.TempDir(), "missing.tmpl")); err == nil {
		t.Fatal("expected error for missing template file")
	}
}
