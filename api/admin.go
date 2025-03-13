package api

import (
	"database/sql"
	"encoding/json"
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

func AdminDeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get admin user ID from session
		adminUserID := middleware.GetUserID(r)
		if adminUserID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify the user is an admin
		adminUser, err := models.GetUserByID(db, int64(adminUserID))
		if err != nil || !adminUser.IsAdmin {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get the user ID to delete
		deleteUserIDString := r.FormValue("userID")
		deleteUserID, err := strconv.ParseInt(deleteUserIDString, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Prevent admins from deleting their own account
		if deleteUserID == int64(adminUserID) {
			http.Error(w, "Admins cannot delete their own account", http.StatusBadRequest)
			return
		}

		// Verify the confirmation text
		confirmText := r.FormValue("confirmText")
		if confirmText != "DELETE" {
			http.Error(w, "Invalid confirmation text", http.StatusBadRequest)
			return
		}

		// Delete the user and all associated data
		err = models.DeleteUserAndData(db, deleteUserID)
		if err != nil {
			http.Error(w, "Error deleting user", http.StatusInternalServerError)
			return
		}

		// Return success
		w.WriteHeader(http.StatusOK)
	}
}

// ToggleSignupStatusHandler handles toggling the signup status
func ToggleSignupStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the request body
		var requestData struct {
			AllowSignups bool `json:"allow_signups"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update the signup status
		err = models.SetSignupStatus(db, requestData.AllowSignups)
		if err != nil {
			log.Printf("Error updating signup status: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"message":       "Signup status updated successfully",
			"allow_signups": requestData.AllowSignups,
		})
	}
}
