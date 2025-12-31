package generator

import "github.com/epilande/codegrab/internal/secrets"

// FileData holds file content for the generated sections
type FileData struct {
	Path     string
	Content  string
	Language string
	Findings []secrets.Finding
}

// TemplateData is injected into the templates
type TemplateData struct {
	Structure string
	Files     []FileData
	FilePaths []string
}

// Format defines the interface for different output formats
type Format interface {
	// Render converts the template data into the specific format
	Render(data TemplateData) (string, int, error)
	// Extension returns the file extension for this format
	Extension() string
	// Name returns the name of the format
	Name() string
}
