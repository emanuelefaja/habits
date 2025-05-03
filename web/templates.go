package web

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// TemplateFuncMap returns the template functions map used across the application
func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"times": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i
			}
			return result
		},
		"add": func(a, b int) int {
			return a + b
		},
		"dict": Dict,
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"safeURL": func(u string) template.URL {
			return template.URL(u)
		},
	}
}

// LoadTemplates loads and parses all application templates
func LoadTemplates() (*template.Template, error) {
	t := template.New("").Funcs(TemplateFuncMap())
	templateCount := 0

	// Walk the ui directory
	if err := filepath.Walk("ui", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .html files
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			// Parse each file
			t, err = t.ParseFiles(path)
			if err != nil {
				log.Printf("Warning: Error parsing template %s: %v", path, err)
				return nil // Continue despite errors
			}
			templateCount++
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Log a concise summary of loaded templates
	log.Printf("✅ Loaded %d/%d templates successfully", templateCount, len(t.Templates()))

	return t, nil
}
