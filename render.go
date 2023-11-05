package coco

import (
	"html/template"
	"io/fs"

	"github.com/noelukwa/tempest"
)

// TemplateConfig is a configuration for loading templates from an fs.FS
type TemplateConfig struct {
	// The file extension of the templates.
	// Defaults to ".html".
	Ext string

	// The directory where the includes are stored.
	// Defaults to "includes".
	IncludesDir string

	// The name used for layout templates :- templates that wrap other contents.
	// Defaults to "layouts".
	Layout string
}

// LoadTemplates loads templates from an fs.FS with a given config
func (a *App) LoadTemplates(fs fs.FS, config *TemplateConfig) (err error) {

	if a.templates == nil {
		a.templates = make(map[string]*template.Template)
	}

	var tmpst *tempest.Tempest
	if config != nil {
		tmpst = tempest.WithConfig(&tempest.Config{
			Layout:      config.Layout,
			IncludesDir: config.IncludesDir,
			Ext:         config.Ext,
		})
	} else {
		tmpst = tempest.New()
	}

	a.templates, err = tmpst.LoadFS(fs)
	return err
}
