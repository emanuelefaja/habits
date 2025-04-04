package web

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
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
			if err := templates.ExecuteTemplate(tw, "guest-home.html", nil); err != nil {
				// Check if the error is due to a client disconnection
				if strings.Contains(err.Error(), "write: broken pipe") ||
					strings.Contains(err.Error(), "client disconnected") ||
					strings.Contains(err.Error(), "connection reset by peer") ||
					strings.Contains(err.Error(), "response timeout exceeded") {
					log.Printf("Client disconnected while rendering guest-home.html: %v", err)
					return
				}
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
		}{
			User:       user,
			HabitsJSON: template.JS(habitsJSON),
			Flash:      middleware.GetFlash(r),
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

// Helper functions for handlers
func renderGuestHome(w http.ResponseWriter, templates *template.Template) {
	if err := templates.ExecuteTemplate(w, "guest-home.html", nil); err != nil {
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
