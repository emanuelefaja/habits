package web

import (
	"encoding/json"
	"html/template"
	"log"
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
	parsedTemplates, err := template.New("").Funcs(TemplateFuncMap()).ParseFiles(
		// Components
		"ui/components/header.html",
		"ui/components/habit-modal.html",
		"ui/components/monthly-grid.html",
		"ui/components/demo-grid.html",
		"ui/components/welcome.html",
		"ui/components/yearly-grid.html",
		"ui/components/head.html",
		"ui/components/footer.html",
		"ui/components/sum-line-graph.html",
		"ui/components/goal.html",
		"ui/components/subscription-form.html",
		// Pages
		"ui/home.html",
		"ui/settings.html",
		"ui/login.html",
		"ui/register.html",
		"ui/roadmap.html",
		"ui/habits/habit.html",
		"ui/habits/binary.html",
		"ui/habits/numeric.html",
		"ui/habits/choice.html",
		"ui/habits/set-rep.html",
		"ui/about.html",
		"ui/guest-home.html",
		"ui/admin.html",
		"ui/changelog.html",
		"ui/blog/blog.html",
		"ui/blog/post.html",
		"ui/goals.html",
		"ui/forgot.html",
		"ui/reset.html",
		"ui/unsubscribe.html",
		"ui/courses/digital-detox.html",
		"ui/courses/phone-addiction.html",
		"ui/privacy.html",
		"ui/terms.html",
		"ui/brand.html",
		"ui/course.html",
	)

	if err != nil {
		return nil, err
	}

	// Log loaded templates
	for _, t := range parsedTemplates.Templates() {
		log.Printf("Loaded template: %s", t.Name())
	}

	return parsedTemplates, nil
}
