package email

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
)

// renderTemplates loads and renders both HTML and text templates for an email
func (s *SMTPEmailService) renderTemplates(templateName string, data interface{}) (htmlContent, textContent string, err error) {
	// Load HTML template
	htmlPath := filepath.Join(s.config.TemplateDir, templateName+".html")
	htmlTmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to load HTML template: %w", err)
	}

	// Load text template
	textPath := filepath.Join(s.config.TemplateDir, templateName+".txt")
	textTmpl, err := template.ParseFiles(textPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to load text template: %w", err)
	}

	// Render both templates
	htmlBuf := new(bytes.Buffer)
	textBuf := new(bytes.Buffer)

	if err := htmlTmpl.Execute(htmlBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render HTML template: %w", err)
	}
	if err := textTmpl.Execute(textBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}
