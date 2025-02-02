package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"mad/middleware"
	"mad/models"

	"golang.org/x/crypto/bcrypt"
)

func AdminResetPasswordHandler(db *sql.DB) http.HandlerFunc {
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

		updateUserIDString := r.FormValue("userID")
		newPassword := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		log.Printf("User ID %v %v %v", updateUserIDString, newPassword, confirmPassword)
		updateUserID, err := strconv.ParseInt(updateUserIDString, 10, 64)
		if err != nil {
			log.Printf("Error %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Validate passwords match
		if newPassword != confirmPassword {
			middleware.SetFlash(r, "New passwords do not match")
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}

		// Update password
		err = models.AdminUpdateUserPassword(db, updateUserID, newPassword)
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
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}
