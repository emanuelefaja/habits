package web

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"mad/api"
	"mad/middleware"
	"mad/models"
)

// HomeHandler handles the home page route
func HomeHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Wrap with timeout
		tw := newTimeoutResponseWriter(w, 10*time.Second)

		if r.URL.Path != "/" {
			http.NotFound(tw, r)
			return
		}

		if !middleware.IsAuthenticated(r) {
			// Guest handler
			data := struct {
				User *models.User
				Page string
			}{
				User: nil,
				Page: "guest-home",
			}
			if err := templates.ExecuteTemplate(tw, "guest-home.html", data); err != nil {
				// Check if the error is due to a client disconnection
				if strings.Contains(err.Error(), "write: broken pipe") ||
					strings.Contains(err.Error(), "client disconnected") ||
					strings.Contains(err.Error(), "connection reset by peer") ||
					strings.Contains(err.Error(), "response timeout exceeded") {
					log.Printf("Client disconnected while rendering guest-home.html: %v", err)
					return
				}
				log.Printf("Error rendering guest-home.html: %v", err)
				http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Authenticated user handling
		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			log.Printf("Error getting authenticated user: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only get necessary habit data for the current view
		habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
		if err != nil {
			log.Printf("Error getting habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Limit the amount of data sent to the template
		habitsJSON, err := json.Marshal(habits)
		if err != nil {
			log.Printf("Error marshaling habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			User       *models.User
			HabitsJSON template.JS
			Flash      string
			Page       string
		}{
			User:       user,
			HabitsJSON: template.JS(habitsJSON),
			Flash:      middleware.GetFlash(r),
			Page:       "home",
		}
		renderTemplate(tw, templates, "home.html", data)
	}
}

// LoginHandler handles the login page
func LoginHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			data := TemplateData{
				Flash: middleware.GetFlash(r),
			}

			// Add quote to the template data
			templateData := map[string]interface{}{
				"Flash": data.Flash,
				"Error": data.Error,
				"Quote": quote,
			}

			renderTemplate(w, templates, "login.html", templateData)
		case http.MethodPost:
			api.LoginHandler(db, templates)(w, r)
		default:
			HandleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}
}

// LogoutHandler handles the logout functionality
func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			HandleNotAllowed(w, http.MethodPost)
			return
		}
		if err := middleware.ClearSession(r); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		middleware.SetFlash(r, "You have been logged out successfully!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// SettingsHandler handles the settings page route
func SettingsHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Debug: Print user settings
		log.Printf("User settings: confetti=%v, weekdays=%v, notifications=%v", user.ShowConfetti, user.ShowWeekdays, user.NotificationEnabled)

		data := struct {
			User  *models.User
			Flash string
		}{
			User:  user,
			Flash: middleware.GetFlash(r),
		}
		renderTemplate(w, templates, "settings.html", data)
	}
}

// BrandHandler handles the brand guidelines page route
func BrandHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Page  string
			User  *models.User
			Flash string
		}{
			Page:  "brand",
			Flash: middleware.GetFlash(r),
		}

		// If user is authenticated, get user data
		if middleware.IsAuthenticated(r) {
			user, err := getAuthenticatedUser(r, db)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			data.User = user
		}

		renderTemplate(w, templates, "brand.html", data)
	}
}

// AboutHandler handles the about page route
func AboutHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "about",
		}
		renderTemplate(w, templates, "about.html", data)
	}
}

// PrivacyHandler handles the privacy policy page route
func PrivacyHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)

		data := map[string]interface{}{
			"User":        user,
			"LastUpdated": time.Now().Format("January 2, 2006"),
		}
		renderTemplate(w, templates, "privacy.html", data)
	}
}

// TermsHandler handles the terms of service page route
func TermsHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the user if logged in
		user, _ := getAuthenticatedUser(r, db)

		data := map[string]interface{}{
			"User":        user,
			"LastUpdated": time.Now().Format("January 2, 2006"),
		}
		renderTemplate(w, templates, "terms.html", data)
	}
}

// BlogHandler handles the blog pages
func BlogHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/blog")
		blogService := models.GetBlogService()

		user, _ := getAuthenticatedUser(r, db)

		if path == "" || path == "/" {
			posts := blogService.GetAllPosts()
			data := struct {
				User  *models.User
				Posts []*models.BlogPost
				Page  string
			}{
				User:  user,
				Posts: posts,
				Page:  "blog",
			}
			renderTemplate(w, templates, "blog.html", data)
			return
		}

		slug := strings.TrimPrefix(path, "/")
		post, exists := blogService.GetPost(slug)
		if !exists {
			http.NotFound(w, r)
			return
		}

		data := struct {
			User *models.User
			Post *models.BlogPost
			Page string
		}{
			User: user,
			Post: post,
			Page: "blog",
		}
		renderTemplate(w, templates, "post.html", data)
	}
}

// ChangelogHandler handles the changelog page route
func ChangelogHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "changelog",
		}
		renderTemplate(w, templates, "changelog.html", data)
	}
}

// RoadmapHandler handles the roadmap page route
func RoadmapHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "roadmap",
		}
		renderTemplate(w, templates, "roadmap.html", data)
	}
}

// RegisterHandler handles the registration page and form submission
func RegisterHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Check if signups are allowed
			allowSignups, err := models.GetSignupStatus(db)
			if err != nil {
				log.Printf("Error checking signup status: %v", err)
				// Default to allowing signups if there's an error
			} else if !allowSignups {
				// Redirect to login page with a message
				middleware.SetFlash(r, "Registration is currently disabled ❌")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Generate math problem for human verification
			num1 := rand.Intn(20) + 1 // Random number between 1-20
			num2 := rand.Intn(20) + 1 // Random number between 1-20
			sum := num1 + num2

			// Store in session
			middleware.SetMathProblem(r, num1, num2, sum)

			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			// Pass to template
			data := map[string]interface{}{
				"MathNum1": num1,
				"MathNum2": num2,
				"Quote":    quote,
			}

			renderTemplate(w, templates, "register.html", data)
		case http.MethodPost:
			api.RegisterHandler(db, templates)(w, r)
		default:
			HandleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}
}

// ForgotPasswordHandler handles the forgot password page
func ForgotPasswordHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleNotAllowed(w, http.MethodGet)
			return
		}

		// Get a random quote
		quote, err := models.GetRandomQuote()
		if err != nil {
			log.Printf("Error getting random quote: %v", err)
			// Continue with default quote from the function
		}

		data := TemplateData{
			IsLoggedIn: middleware.IsAuthenticated(r),
		}
		if data.IsLoggedIn {
			user, err := getAuthenticatedUser(r, db)
			if err == nil {
				data.Email = user.Email
			}
		}

		// Add quote to the template data
		templateData := map[string]interface{}{
			"IsLoggedIn": data.IsLoggedIn,
			"Email":      data.Email,
			"Quote":      quote,
		}

		renderTemplate(w, templates, "forgot.html", templateData)
	}
}

// ResetPasswordHandler handles the password reset page
func ResetPasswordHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleNotAllowed(w, http.MethodGet)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Redirect(w, r, "/forgot", http.StatusSeeOther)
			return
		}

		// Get a random quote
		quote, err := models.GetRandomQuote()
		if err != nil {
			log.Printf("Error getting random quote: %v", err)
			// Continue with default quote from the function
		}

		// Add quote to the template data
		templateData := map[string]interface{}{
			"Token": token,
			"Flash": middleware.GetFlash(r),
			"Quote": quote,
		}

		renderTemplate(w, templates, "reset.html", templateData)
	}
}

// Helper functions for handlers
func renderGuestHome(w http.ResponseWriter, templates *template.Template) {
	data := struct {
		User *models.User
		Page string
	}{
		User: nil,
		Page: "guest-home",
	}
	if err := templates.ExecuteTemplate(w, "guest-home.html", data); err != nil {
		handleTemplateError(w, err, "guest-home.html")
	}
}

func renderUserHome(w http.ResponseWriter, r *http.Request, db *sql.DB, templates *template.Template) {
	user, err := getAuthenticatedUser(r, db)
	if err != nil {
		log.Printf("Error getting authenticated user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
	if err != nil {
		log.Printf("Error getting habits: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	habitsJSON, err := json.Marshal(habits)
	if err != nil {
		log.Printf("Error marshaling habits: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		User       *models.User
		HabitsJSON template.JS
		Flash      string
	}{
		User:       user,
		HabitsJSON: template.JS(habitsJSON),
		Flash:      middleware.GetFlash(r),
	}

	renderTemplate(w, templates, "home.html", data)
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, templates *template.Template) {
	quote, err := models.GetRandomQuote()
	if err != nil {
		log.Printf("Error getting random quote: %v", err)
	}

	templateData := map[string]interface{}{
		"Flash": middleware.GetFlash(r),
		"Quote": quote,
	}

	renderTemplate(w, templates, "login.html", templateData)
}

func handleTemplateError(w http.ResponseWriter, err error, templateName string) {
	if strings.Contains(err.Error(), "write: broken pipe") ||
		strings.Contains(err.Error(), "client disconnected") ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "response timeout exceeded") {
		log.Printf("Client disconnected while rendering %s: %v", templateName, err)
		return
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
