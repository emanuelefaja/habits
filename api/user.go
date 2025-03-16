package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mad/middleware"
	"mad/models"
	"mad/models/email"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Global email service
var emailService email.EmailService

// InitEmailService initializes the email service for the API package
func InitEmailService(service email.EmailService) {
	emailService = service
}

// RegisterHandler handles user registration
func RegisterHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if signups are allowed
		allowSignups, err := models.GetSignupStatus(db)
		if err != nil {
			log.Printf("Error checking signup status: %v", err)
			// Default to allowing signups if there's an error
		} else if !allowSignups {
			log.Printf("Registration attempt when signups are disabled")
			w.WriteHeader(http.StatusForbidden)
			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error": "Registration is currently disabled ❌",
			})
			return
		}

		// Get the real IP address using our helper function
		ip := middleware.GetClientIP(r)
		log.Printf("Registration attempt from IP: %s", ip)

		// Check IP rate limiting - pass the original request
		remaining, resetTime, err := middleware.RegistrationLimiter.CheckLimit(r)
		if err != nil || remaining <= 0 {
			log.Printf("Registration rate limit exceeded for IP: %s. Reset at: %v", ip, resetTime)
			w.WriteHeader(http.StatusTooManyRequests)
			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error": "Too many registration attempts. Please try again later ⏳",
			})
			return
		}
		log.Printf("Registration attempt allowed for IP: %s. Remaining attempts: %d", ip, remaining)

		// Validate math problem answer
		mathAnswer := r.FormValue("math_answer")
		if mathAnswer == "" {
			log.Println("Math answer is missing")
			w.WriteHeader(http.StatusBadRequest)

			// Generate new math problem
			num1 := rand.Intn(20) + 1
			num2 := rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Please answer the math verification question ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		// Convert answer to int
		userAnswer, err := strconv.Atoi(mathAnswer)
		if err != nil {
			log.Println("Invalid math answer format:", err)
			w.WriteHeader(http.StatusBadRequest)

			// Generate new math problem
			num1 := rand.Intn(20) + 1
			num2 := rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Invalid answer format ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		// Get expected answer from session
		num1, num2, expectedSum, ok := middleware.GetMathProblem(r)
		if !ok {
			log.Println("Math problem not found in session")
			w.WriteHeader(http.StatusBadRequest)

			// Generate new math problem
			num1 = rand.Intn(20) + 1
			num2 = rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Session expired, please try again ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		// Validate answer
		if userAnswer != expectedSum {
			log.Printf("Incorrect math answer: got %d, expected %d", userAnswer, expectedSum)
			w.WriteHeader(http.StatusBadRequest)

			// Generate new math problem
			num1 = rand.Intn(20) + 1
			num2 = rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Incorrect answer. Please try again ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		// Clear math problem from session after successful validation
		middleware.ClearMathProblem(r)

		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		log.Println("Received registration request for:", email)

		// Basic validation
		if firstName == "" || lastName == "" || email == "" || password == "" {
			log.Println("Validation failed: missing fields")
			w.WriteHeader(http.StatusBadRequest)

			// Generate new math problem
			num1 = rand.Intn(20) + 1
			num2 = rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "All fields are required ❌",
				"MathNum1": num1,
				"MathNum2": num2,
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
			w.WriteHeader(http.StatusInternalServerError)

			// Generate new math problem
			num1 := rand.Intn(20) + 1
			num2 := rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Internal server error ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		err = user.Create(db, string(passwordHash))
		if err != nil {
			log.Println("Error creating user:", err)
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				w.WriteHeader(http.StatusConflict)

				// Generate new math problem
				num1 := rand.Intn(20) + 1
				num2 := rand.Intn(20) + 1
				sum := num1 + num2
				middleware.SetMathProblem(r, num1, num2, sum)

				tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
					"Error":    "This email is already registered ✉️",
					"MathNum1": num1,
					"MathNum2": num2,
				})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)

			// Generate new math problem
			num1 := rand.Intn(20) + 1
			num2 := rand.Intn(20) + 1
			sum := num1 + num2
			middleware.SetMathProblem(r, num1, num2, sum)

			tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
				"Error":    "Error creating account ❌",
				"MathNum1": num1,
				"MathNum2": num2,
			})
			return
		}

		log.Println("User registered successfully:", email)

		// Send welcome email
		if emailService != nil {
			go func() {
				err := emailService.SendWelcomeEmail(email, firstName)
				if err != nil {
					log.Printf("Failed to send welcome email to %s: %v", email, err)
				} else {
					log.Printf("Welcome email sent to %s", email)
				}
			}()
		}

		// Set user session immediately after registration
		middleware.SetUserID(r, int(user.ID))

		// Return success response for the fetch request
		w.WriteHeader(http.StatusOK)
	}
}

// LoginHandler handles user login
func LoginHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get IP address from request using our helper function
		ip := middleware.GetClientIP(r)
		log.Printf("Login attempt from IP: %s", ip)

		// Check if IP is blocked - use the original request
		remaining, resetTime, err := middleware.LoginLimiter.CheckLimit(r)
		if err != nil || remaining <= 0 {
			log.Printf("Login rate limit exceeded for IP: %s. Reset at: %v", ip, resetTime)
			tmpl.ExecuteTemplate(w, "login.html", TemplateData{
				Error: "Too many login attempts. Please try again later ⏳",
			})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		// First validate the password
		valid, err := models.ValidatePassword(db, email, password)
		if err != nil || !valid {
			// Record failed attempt and check remaining attempts - use the original request
			remaining, resetTime, _ := middleware.LoginLimiter.CheckLimit(r)
			if remaining <= 0 {
				log.Printf("Login rate limit exceeded after failed attempt for IP: %s. Reset at: %v", ip, resetTime)
				tmpl.ExecuteTemplate(w, "login.html", TemplateData{
					Error: "Too many login attempts. Please try again later ⏳",
				})
				return
			}

			tmpl.ExecuteTemplate(w, "login.html", TemplateData{
				Error: fmt.Sprintf("Invalid email or password ❌ (%d attempts remaining)", remaining),
			})
			return
		}

		// If login successful, get the user
		if valid {
			// Get user for the session
			if user, err := models.GetUserByEmail(db, email); err == nil {
				// Set user session
				middleware.SetUserID(r, int(user.ID))

				// Set welcome flash message
				middleware.SetFlash(r, "Welcome back, "+user.FirstName+"! ✨")

				// Redirect to home page
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		// If we get here, something went wrong
		tmpl.ExecuteTemplate(w, "login.html", TemplateData{
			Error: "An error occurred while logging in ❌",
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

// ExportDataHandler generates and serves a CSV file of the user's habits and logs
func ExportDataHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Set headers for CSV download
		filename := fmt.Sprintf("habits_export_%s.csv", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		// Create CSV writer
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		// Write headers
		headers := []string{"Habit Name", "Emoji", "Type", "Date", "Status", "Value"}
		if err := csvWriter.Write(headers); err != nil {
			http.Error(w, "Error writing CSV headers", http.StatusInternalServerError)
			return
		}

		// Get all habits for the user
		habits, err := models.GetHabitsByUserID(db, userID)
		if err != nil {
			http.Error(w, "Error fetching habits", http.StatusInternalServerError)
			return
		}

		// For each habit, get its logs
		for _, habit := range habits {
			// Get logs for the past year
			endDate := time.Now()
			startDate := endDate.AddDate(-1, 0, 0)
			logs, err := models.GetHabitLogsByDateRange(db, habit.ID, startDate, endDate)
			if err != nil {
				continue // Skip this habit if there's an error
			}

			// Write habit logs
			for _, log := range logs {
				var value string
				if log.Value.Valid {
					if habit.HabitType == models.NumericHabit {
						// For numeric habits, extract just the number
						var valueMap map[string]interface{}
						if err := json.Unmarshal([]byte(log.Value.String), &valueMap); err == nil {
							if numValue, ok := valueMap["value"]; ok {
								value = fmt.Sprintf("%v", numValue)
							}
						}
					} else {
						// For other habit types, keep the full JSON
						value = log.Value.String
					}
				}

				row := []string{
					habit.Name,
					habit.Emoji,
					string(habit.HabitType),
					log.Date.Format("2006-01-02"),
					log.Status,
					value,
				}
				if err := csvWriter.Write(row); err != nil {
					continue // Skip this row if there's an error
				}
			}
		}
	}
}

// UpdateSettingsHandler handles updating user settings
func UpdateSettingsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from session
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse JSON request
		var settings struct {
			ShowConfetti        bool `json:"showConfetti"`
			ShowWeekdays        bool `json:"showWeekdays"`
			NotificationEnabled bool `json:"notificationEnabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			log.Printf("Error decoding settings JSON: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Updating settings for user %d: confetti=%v, weekdays=%v, notifications=%v",
			userID, settings.ShowConfetti, settings.ShowWeekdays, settings.NotificationEnabled)

		// Update settings in database
		result, err := db.Exec(`
			UPDATE users 
			SET show_confetti = ?, show_weekdays = ?, notification_enabled = ?
			WHERE id = ?
		`, settings.ShowConfetti, settings.ShowWeekdays, settings.NotificationEnabled, userID)

		if err != nil {
			log.Printf("Error updating settings in database: %v", err)
			http.Error(w, "Error updating settings", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("Settings update affected %d rows", rowsAffected)

		// Verify the update by retrieving the user again
		var updatedUser struct {
			ShowConfetti        bool
			ShowWeekdays        bool
			NotificationEnabled bool
		}
		err = db.QueryRow(`
			SELECT show_confetti, show_weekdays, notification_enabled
			FROM users
			WHERE id = ?
		`, userID).Scan(&updatedUser.ShowConfetti, &updatedUser.ShowWeekdays, &updatedUser.NotificationEnabled)

		if err != nil {
			log.Printf("Error verifying settings update: %v", err)
		} else {
			log.Printf("User %d settings after update: confetti=%v, weekdays=%v, notifications=%v",
				userID, updatedUser.ShowConfetti, updatedUser.ShowWeekdays, updatedUser.NotificationEnabled)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

// ResetDataHandler handles resetting all user data
func ResetDataHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from session
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Reset user data
		err := models.ResetUserData(db, int64(userID))
		if err != nil {
			log.Printf("Error resetting user data: %v", err)
			middleware.SetFlash(r, "Error resetting data ❌")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		// Set success flash message
		middleware.SetFlash(r, "All habit data has been reset successfully ✨")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// UpdateNotificationPreferenceHandler handles updating the user's notification preference
func UpdateNotificationPreferenceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get user ID from session
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse JSON request
		var request struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update notification preference in database
		err := models.UpdateNotificationPreference(db, int64(userID), request.Enabled)
		if err != nil {
			log.Printf("Error updating notification preference: %v", err)
			http.Error(w, "Error updating notification preference", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Notification preference updated successfully",
			"enabled": request.Enabled,
		})
	}
}
