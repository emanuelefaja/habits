package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"mad/middleware"
	"mad/models"
)

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Message           string `json:"message"`
	RemainingAttempts int    `json:"remaining_attempts,omitempty"`
	BlockDuration     string `json:"block_duration,omitempty"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type ResetPasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// generateResetToken creates a secure random token
func generateResetToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(token), nil
}

// ForgotPasswordHandler handles password reset requests
func ForgotPasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("🔑 Starting password reset request handling")

		if r.Method != http.MethodPost {
			log.Println("❌ Invalid method:", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ForgotPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println("❌ Failed to decode request body:", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		log.Printf("📧 Processing reset request for email: %s", req.Email)

		// Check rate limiting
		remaining, blockedUntil, err := middleware.PasswordResetLimiter.CheckLimit(r)
		if err != nil {
			log.Printf("❌ Rate limit check failed: %v", err)
			http.Error(w, "Too many attempts", http.StatusTooManyRequests)
			return
		}

		if remaining <= 0 {
			log.Printf("⛔ Rate limit exceeded. Blocked until: %v", blockedUntil)
			respondJSON(w, ForgotPasswordResponse{
				Message:       "Too many attempts. Please try again later.",
				BlockDuration: blockedUntil.Sub(time.Now()).String(),
			})
			return
		}
		log.Printf("✅ Rate limit check passed. Remaining attempts: %d", remaining)

		// Get user by email
		user, err := models.GetUserByEmail(db, req.Email)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("⚠️ No user found with email: %s", req.Email)
				// Don't reveal if email exists
				respondJSON(w, ForgotPasswordResponse{
					Message:           "If an account exists with this email, you will receive password reset instructions.",
					RemainingAttempts: remaining,
				})
				return
			}
			log.Printf("❌ Database error looking up user: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Printf("✅ Found user: %s (ID: %d)", user.Email, user.ID)

		// Generate reset token
		token, err := generateResetToken()
		if err != nil {
			log.Printf("❌ Failed to generate reset token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Println("✅ Generated reset token")

		// Set token expiry to 1 hour from now
		expiry := time.Now().Add(time.Hour)

		// Invalidate any existing tokens for this user
		if err := models.InvalidateExistingTokens(db, user.Email); err != nil {
			log.Printf("❌ Failed to invalidate existing tokens: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Println("✅ Invalidated existing tokens")

		// Store new token
		if err := models.CreateResetToken(db, user.ID, user.Email, token, expiry); err != nil {
			log.Printf("❌ Failed to store reset token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Println("✅ Stored new reset token")

		// Send reset email
		resetLink := "https://" + r.Host + "/reset?token=" + token
		log.Printf("🔗 Generated reset link: %s", resetLink)

		if err := emailService.SendPasswordResetEmail(user.Email, resetLink, expiry); err != nil {
			log.Printf("❌ Failed to send reset email: %v", err)
			// Log error but don't reveal to user
			respondJSON(w, ForgotPasswordResponse{
				Message:           "If an account exists with this email, you will receive password reset instructions.",
				RemainingAttempts: remaining,
			})
			return
		}
		log.Printf("✅ Successfully sent reset email to: %s", user.Email)

		respondJSON(w, ForgotPasswordResponse{
			Message:           "If an account exists with this email, you will receive password reset instructions.",
			RemainingAttempts: remaining,
		})
		log.Println("✅ Password reset request completed successfully")
	}
}

// ResetPasswordHandler handles password reset with token
func ResetPasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ResetPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate password
		if len(req.Password) < 8 {
			respondJSON(w, ResetPasswordResponse{
				Success: false,
				Message: "Password must be at least 8 characters long",
			})
			return
		}

		// Get and validate token
		token, err := models.GetResetToken(db, req.Token)
		if err != nil {
			if err == sql.ErrNoRows {
				respondJSON(w, ResetPasswordResponse{
					Success: false,
					Message: "Invalid or expired reset token",
				})
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if token.Used {
			respondJSON(w, ResetPasswordResponse{
				Success: false,
				Message: "This reset token has already been used",
			})
			return
		}

		if time.Now().After(token.Expiry) {
			respondJSON(w, ResetPasswordResponse{
				Success: false,
				Message: "Reset token has expired",
			})
			return
		}

		// Hash new password
		hashedPassword, err := models.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Update password and mark token as used
		if err := models.UpdateUserPassword(db, token.UserID, hashedPassword); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := models.MarkTokenUsed(db, req.Token); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Send success email
		if err := emailService.SendPasswordResetSuccessEmail(token.Email, token.Username); err != nil {
			// Log error but don't reveal to user
		}

		respondJSON(w, ResetPasswordResponse{
			Success: true,
			Message: "Password has been reset successfully",
		})
	}
}
