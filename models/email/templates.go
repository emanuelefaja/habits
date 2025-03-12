package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
)

// renderTemplates loads and renders both HTML and text templates for an email
func (s *SMTPEmailService) renderTemplates(templateName string, data interface{}) (htmlContent, textContent string, err error) {
	log.Printf("🎨 Rendering email templates for: %s", templateName)

	// Load HTML template
	htmlPath := filepath.Join(s.config.TemplateDir, templateName+".html")
	log.Printf("📄 Loading HTML template from: %s", htmlPath)

	htmlTmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		log.Printf("❌ Failed to load HTML template: %v", err)
		return "", "", fmt.Errorf("failed to load HTML template: %w", err)
	}

	// Load text template
	textPath := filepath.Join(s.config.TemplateDir, templateName+".txt")
	log.Printf("📄 Loading text template from: %s", textPath)

	textTmpl, err := template.ParseFiles(textPath)
	if err != nil {
		log.Printf("❌ Failed to load text template: %v", err)
		return "", "", fmt.Errorf("failed to load text template: %w", err)
	}

	// Render both templates
	htmlBuf := new(bytes.Buffer)
	textBuf := new(bytes.Buffer)

	if err := htmlTmpl.Execute(htmlBuf, data); err != nil {
		log.Printf("❌ Failed to render HTML template: %v", err)
		return "", "", fmt.Errorf("failed to render HTML template: %w", err)
	}
	if err := textTmpl.Execute(textBuf, data); err != nil {
		log.Printf("❌ Failed to render text template: %v", err)
		return "", "", fmt.Errorf("failed to render text template: %w", err)
	}

	log.Printf("✅ Successfully rendered both templates for: %s", templateName)
	return htmlBuf.String(), textBuf.String(), nil
}
