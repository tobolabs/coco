package coco

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplates(t *testing.T) {
	// Create a temporary directory for the templates
	tmpDir, err := os.MkdirTemp("", "coco-templates")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create template files with content
	templateFiles := map[string]string{
		"template1.html": "Template 1 Content",
		"template2.html": "Template 2 Content",
		"template3.html": "Template 3 Content",
	}

	// Write template content to the files
	for filename, content := range templateFiles {
		templatePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
			t.Fatalf("Error writing template file %s: %v", filename, err)
		}
	}

	app := NewApp()
	config := &TemplateConfig{
		Ext:         ".html",
		IncludesDir: "includes",
		Layout:      "layouts",
	}

	// Load templates with custom configuration
	err = app.LoadTemplates(NewTestFS(tmpDir), config)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if app.templates == nil {
		t.Fatal("templates map was not initialized")
	}

	// Check if default values are used
	if app.templates["template1"] == nil {
		t.Errorf("Expected template1.html to be loaded, but it was not")
	}

	if app.templates["template2"] == nil {
		t.Errorf("Expected template2.html to be loaded, but it was not")
	}

	if app.templates["template3"] == nil {
		t.Errorf("Expected template3.html to be loaded, but it was not")
	}

}

// NewTestFS creates a fs.FS from the given directory path.
func NewTestFS(dirPath string) fs.FS {
	return os.DirFS(dirPath)
}
