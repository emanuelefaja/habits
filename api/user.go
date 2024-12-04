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

		// Set user session immediately after registration
		middleware.SetUserID(r, int(user.ID))

		// Set welcome flash message
		middleware.SetFlash(r, "Welcome to Habits, "+user.FirstName+"! 🎉")

		// Redirect to home page instead of login
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

// UpdateProfileHandler handles updating user profile information
func UpdateProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from session
		userID := middleware.GetUserID(r)
		if userID == 0 {
			middleware.SetFlash(r, "Session expired. Please login again.")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Get current user
		user, err := models.GetUserByID(db, int64(userID))
		if err != nil {
			middleware.SetFlash(r, "Error finding user.")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		// Update user information
		user.FirstName = r.FormValue("first_name")
		user.LastName = r.FormValue("last_name")
		user.Email = r.FormValue("email")

		if err := user.Update(db); err != nil {
			// Check if error is due to duplicate email
			if err.Error() == "UNIQUE constraint failed: users.email" {
				middleware.SetFlash(r, "Email already in use by another account.")
				http.Redirect(w, r, "/settings", http.StatusSeeOther)
				return
			}
			middleware.SetFlash(r, "Error updating profile.")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		middleware.SetFlash(r, "Profile updated successfully! ✨")
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

// UpdatePasswordHandler handles password updates
func UpdatePasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate passwords match
		if newPassword != confirmPassword {
			middleware.SetFlash(r, "New passwords do not match")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		// Update password
		err := models.UpdatePassword(db, int64(userID), currentPassword, newPassword)
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				middleware.SetFlash(r, "Current password is incorrect")
			} else {
				middleware.SetFlash(r, "Error updating password")
			}
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		middleware.SetFlash(r, "Password updated successfully!")
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

// DeleteAccountHandler handles account deletion
func DeleteAccountHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Delete user and all associated data
		err := models.DeleteUserAndData(db, int64(userID))
		if err != nil {
			middleware.SetFlash(r, "Error deleting account")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		// Clear the session
		middleware.ClearSession(r)

		// Redirect to login page with success message
		middleware.SetFlash(r, "Account deleted successfully")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
