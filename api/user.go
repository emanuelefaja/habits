package api

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"mad/middleware"
	"mad/models"

	"golang.org/x/crypto/bcrypt"
)

// RegisterHandler handles user registration
func RegisterHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		log.Println("Received registration request for:", email)

		// Basic validation
		if firstName == "" || lastName == "" || email == "" || password == "" {
			log.Println("Validation failed: missing fields")
			tmpl.ExecuteTemplate(w, "register.html", TemplateData{
				Error: "All fields are required",
			})
			return
		}

		// Create user
		user := &models.User{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Error generating password hash:", err)
			tmpl.ExecuteTemplate(w, "register.html", TemplateData{
				Error: "Internal server error",
			})
			return
		}

		err = user.Create(db, string(passwordHash))
		if err != nil {
			log.Println("Error creating user:", err)
			tmpl.ExecuteTemplate(w, "register.html", TemplateData{
				Error: "Email already in use",
			})
			return
		}

		log.Println("User registered successfully:", email)

		// Set flash message for successful registration
		middleware.SetFlash(r, "Registration successful! Please log in.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// LoginHandler handles user login
func LoginHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		// First validate the password
		valid, err := models.ValidatePassword(db, email, password)
		if err != nil || !valid {
			tmpl.ExecuteTemplate(w, "login.html", TemplateData{
				Error: "Invalid email or password",
			})
			return
		}

		// If password is valid, get the user
		if valid {
			// Get user for the session
			if user, err := models.GetUserByEmail(db, email); err == nil {
				// Set user session
				middleware.SetUserID(r, int(user.ID))

				// Set welcome flash message
				middleware.SetFlash(r, "Welcome back, "+user.FirstName+"!")

				// Redirect to home page
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		// If we get here, something went wrong
		tmpl.ExecuteTemplate(w, "login.html", TemplateData{
			Error: "Internal server error",
		})
	}
}

// TemplateData holds data for template rendering
type TemplateData struct {
	Flash string
	Error string
}
