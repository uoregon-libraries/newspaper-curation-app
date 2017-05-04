package tmpl

import (
	"html/template"
	"path/filepath"
)

// Template wraps html/template's type in order to provide a name so a single
// template can be self-contained
type Template struct {
	*template.Template
	Name string
}

// TRoot wraps template.Template for use to spawn "real" templates.  The TRoot
// is never meant to be directly rendered itself, but a top-level object for
// collecting the template path on disk, a layout template and shared templates
// (e.g., sidebar), and template functions for reuse in renderable templates
type TRoot struct {
	*template.Template
	Name, Path string
}

// Root creates a new TRoot for use in spawning templates
func Root(name, path string, fnList template.FuncMap) *TRoot {
	var t = &TRoot{template.New(name), name, path}
	t.Template.Funcs(fnList)

	return t
}

// ReadPartials parses the given files into the TRoot instance for gathering
// things like the top-level layout, navigation elements, etc.  The list of
// files is relative to the TRoot's Path.  Returns on the first error
// encountered, if any.
func (t *TRoot) ReadPartials(files ...string) error {
	for _, file := range files {
		var _, err = t.Template.ParseFiles(filepath.Join(t.Path, file))
		if err != nil {
			return err
		}
	}

	return nil
}

// MustReadPartials calls ReadPartials and panics on any error
func (t *TRoot) MustReadPartials(files ...string) {
	var err = t.ReadPartials(files...)
	if err != nil {
		panic(err)
	}
}

// Build clones the root (for layout, funcs, etc) and parses the given file in
// the clone.  The returned template is the clone, and is safe to alter without
// worrying about breaking the root.
func (t *TRoot) Build(path string) (*Template, error) {
	var tNew, err = t.Clone()
	if err != nil {
		return nil, err
	}

	tNew, err = tNew.ParseFiles(filepath.Join(t.Path, path))
	if err != nil {
		return nil, err
	}

	return &Template{tNew, path}, nil
}

// MustBuild calls Build and panics on any error
func (t *TRoot) MustBuild(path string) *Template {
	var tmpl, err = t.Build(path)
	if err != nil {
		panic(err)
	}
	return tmpl
}
