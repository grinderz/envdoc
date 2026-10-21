package render

import (
	"fmt"
	"path"
	"path/filepath"

	texttmpl "text/template"
)

// SetTemplateFile renders with the template file instead of the built-in
// template of the format. Option strings still come from the format config,
// and built-in helpers like `item.options` are available to the template.
func (r *Renderer) SetTemplateFile(name string) error {
	t, err := texttmpl.New(filepath.Base(name)).
		Funcs(tplFuncs).
		ParseFS(templatesFS, path.Join(tmplDir, tmplHelpers))
	if err != nil {
		return fmt.Errorf("parse helpers: %w", err)
	}
	if t, err = t.ParseFiles(name); err != nil {
		return fmt.Errorf("parse template %q: %w", name, err)
	}
	r.tmpl = t
	return nil
}
