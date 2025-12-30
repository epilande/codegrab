package formats

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/epilande/codegrab/internal/generator"
	"github.com/epilande/codegrab/internal/utils"
)

// MarkdownFormat implements the generator.Format interface for Markdown output
type MarkdownFormat struct{}

// Render converts the template data into Markdown format
func (f *MarkdownFormat) Render(data generator.TemplateData) (string, int, error) {
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", 0, fmt.Errorf("failed to execute template: %w", err)
	}

	markdownContent := buf.String()
	tokenCount := utils.EstimateTokens(markdownContent)
	return markdownContent, tokenCount, nil
}

// Extension returns the file extension for Markdown
func (f *MarkdownFormat) Extension() string {
	return ".md"
}

// Name returns the name of the format
func (f *MarkdownFormat) Name() string {
	return "markdown"
}

// The base template for our generated markdown
const markdownTemplate = `# Project Structure

` + "```" + `
{{.Structure}}` + "```" + `
{{if .Files}}
# Project Files
{{range .Files}}
## File: ` + "`" + `{{.Path}}` + "`" + `

` + "```" + `{{.Language}}
{{.Content}}
` + "```" + `
{{end}}{{end}}`
