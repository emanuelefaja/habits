package masterclass

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// Dict creates a dictionary for template data
func Dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("odd number of arguments to dict")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// ProcessTemplate executes a template with component support
func ProcessTemplate(templatePath string, data interface{}) (string, error) {
	// Get the list of component templates
	componentsDir := filepath.Join("ui", "masterclass", "components")
	componentFiles, err := filepath.Glob(filepath.Join(componentsDir, "*.html"))
	if err != nil {
		return "", fmt.Errorf("error finding component templates: %w", err)
	}

	// Create a new template set with our function map
	tmpl := template.New("").Funcs(template.FuncMap{
		"dict": Dict,
		"add": func(a, b int) int {
			return a + b
		},
	})

	// Parse all component templates
	if len(componentFiles) > 0 {
		tmpl, err = tmpl.ParseFiles(componentFiles...)
		if err != nil {
			return "", fmt.Errorf("error parsing component templates: %w", err)
		}
	}

	// Read the template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("error reading template file: %w", err)
	}

	// Parse the template content
	tmpl, err = tmpl.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(templatePath), data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}
