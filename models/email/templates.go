package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
	"strings"
)

// Template functions for email templates
var emailTemplatesFuncMap = template.FuncMap{
	"safeURL": func(u string) template.URL {
		return template.URL(u)
	},
}

// renderTemplates loads and renders both HTML and text templates for an email
func (s *SMTPEmailService) renderTemplates(templateName string, data interface{}) (htmlContent, textContent string, err error) {
	log.Printf("🎨 Rendering email templates for: %s", templateName)

	// Check if this is a campaign email (has a prefix like "courses/digital-detox/")
	if strings.Contains(templateName, "/") {
		return s.renderCampaignTemplates(templateName, data)
	}

	// Regular email templates (non-campaign)
	// Load HTML template
	htmlPath := filepath.Join(s.config.TemplateDir, templateName+".html")
	log.Printf("📄 Loading HTML template from: %s", htmlPath)

	htmlTmpl, err := template.New(filepath.Base(htmlPath)).Funcs(emailTemplatesFuncMap).ParseFiles(htmlPath)
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

// renderCampaignTemplates renders templates for campaign emails using the base template
func (s *SMTPEmailService) renderCampaignTemplates(templateName string, data interface{}) (htmlContent, textContent string, err error) {
	log.Printf("📧 Rendering campaign email template: %s", templateName)

	// Campaign template paths
	htmlPath := filepath.Join(s.config.TemplateDir, templateName+".html")
	textPath := filepath.Join(s.config.TemplateDir, templateName+".txt")
	baseHTMLPath := filepath.Join(s.config.TemplateDir, "base.html")
	baseTextPath := filepath.Join(s.config.TemplateDir, "base.txt")

	// Extract the unsubscribe link before rendering for post-processing
	unsubscribeLink := ""
	if linkData, ok := data.(map[string]interface{})["UnsubscribeLink"]; ok {
		if linkStr, ok := linkData.(string); ok {
			unsubscribeLink = linkStr
			log.Printf("📝 Extracted unsubscribe link for post-processing: %s", unsubscribeLink)
		}
	}

	// Load base templates
	baseHTMLTmpl, err := template.New(filepath.Base(baseHTMLPath)).Funcs(emailTemplatesFuncMap).ParseFiles(baseHTMLPath)
	if err != nil {
		log.Printf("❌ Failed to load base HTML template: %v", err)
		return "", "", fmt.Errorf("failed to load base HTML template: %w", err)
	}

	baseTextTmpl, err := template.ParseFiles(baseTextPath)
	if err != nil {
		log.Printf("❌ Failed to load base text template: %v", err)
		return "", "", fmt.Errorf("failed to load base text template: %w", err)
	}

	// Load and render content templates
	contentHTML, err := template.ParseFiles(htmlPath)
	if err != nil {
		log.Printf("❌ Failed to load content HTML template: %v", err)
		return "", "", fmt.Errorf("failed to load content HTML template: %w", err)
	}

	contentText, err := template.ParseFiles(textPath)
	if err != nil {
		log.Printf("❌ Failed to load content text template: %v", err)
		return "", "", fmt.Errorf("failed to load content text template: %w", err)
	}

	// Render content templates first
	contentHTMLBuf := new(bytes.Buffer)
	contentTextBuf := new(bytes.Buffer)

	if err := contentHTML.Execute(contentHTMLBuf, data); err != nil {
		log.Printf("❌ Failed to render content HTML template: %v", err)
		return "", "", fmt.Errorf("failed to render content HTML template: %w", err)
	}

	if err := contentText.Execute(contentTextBuf, data); err != nil {
		log.Printf("❌ Failed to render content text template: %v", err)
		return "", "", fmt.Errorf("failed to render content text template: %w", err)
	}

	// Create base template data with rendered content
	campaignData := map[string]interface{}{
		"Content":       template.HTML(contentHTMLBuf.String()),
		"Title":         data.(map[string]interface{})["Title"],
		"Subject":       data.(map[string]interface{})["Subject"],
		"AppName":       data.(map[string]interface{})["AppName"],
		"CampaignName":  data.(map[string]interface{})["CampaignName"],
		"CampaignEmoji": data.(map[string]interface{})["CampaignEmoji"],
		// Add a placeholder for the unsubscribe link that we'll replace later
		"UnsubscribeLink": "UNSUBSCRIBE_LINK_PLACEHOLDER",
		"FirstName":       data.(map[string]interface{})["FirstName"],
	}

	// Render base templates with content
	finalHTMLBuf := new(bytes.Buffer)
	finalTextBuf := new(bytes.Buffer)

	if err := baseHTMLTmpl.Execute(finalHTMLBuf, campaignData); err != nil {
		log.Printf("❌ Failed to render base HTML template: %v", err)
		return "", "", fmt.Errorf("failed to render base HTML template: %w", err)
	}

	// For text template, use plain text content
	textCampaignData := map[string]interface{}{
		"Content":         contentTextBuf.String(),
		"Title":           data.(map[string]interface{})["Title"],
		"AppName":         data.(map[string]interface{})["AppName"],
		"CampaignName":    data.(map[string]interface{})["CampaignName"],
		"CampaignEmoji":   data.(map[string]interface{})["CampaignEmoji"],
		"UnsubscribeLink": unsubscribeLink, // Use the original link for text emails
	}

	if err := baseTextTmpl.Execute(finalTextBuf, textCampaignData); err != nil {
		log.Printf("❌ Failed to render base text template: %v", err)
		return "", "", fmt.Errorf("failed to render base text template: %w", err)
	}

	// Post-process the HTML to replace the placeholder with the actual unsubscribe link
	// This avoids any HTML encoding by the template engine
	htmlOutput := finalHTMLBuf.String()
	if unsubscribeLink != "" {
		htmlOutput = strings.Replace(htmlOutput, "UNSUBSCRIBE_LINK_PLACEHOLDER", unsubscribeLink, 1)
		log.Printf("📝 Replaced unsubscribe link placeholder in HTML output")
	}

	log.Printf("✅ Successfully rendered campaign templates for: %s", templateName)
	return htmlOutput, finalTextBuf.String(), nil
}
