package middleware

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

var SessionManager *scs.SessionManager

func InitializeSession(db *sql.DB) error {
	// Create sessions table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create a new SQLite session store
	sessionStore := sqlite3store.New(db)

	SessionManager = scs.New()
	SessionManager.Store = sessionStore

	// Configure session settings
	SessionManager.Lifetime = 7 * 24 * time.Hour
	SessionManager.Cookie.Secure = false // Changed to false for development
	SessionManager.Cookie.SameSite = http.SameSiteLaxMode
	SessionManager.Cookie.HttpOnly = true

	return nil
}

// Helper functions for flash messages
func SetFlash(r *http.Request, message string) {
	SessionManager.Put(r.Context(), "flash", message)
}

func GetFlash(r *http.Request) string {
	message := SessionManager.PopString(r.Context(), "flash")
	return message
}

// Authentication helpers
func IsAuthenticated(r *http.Request) bool {
	exists := SessionManager.Exists(r.Context(), "userID")
	log.Printf("IsAuthenticated: Session check result: %v", exists)
	return exists
}

func SetUserID(r *http.Request, userID int) {
	SessionManager.Put(r.Context(), "userID", userID)
}

func GetUserID(r *http.Request) int {
	userID, ok := SessionManager.Get(r.Context(), "userID").(int)
	log.Printf("GetUserID: Retrieved userID: %v, ok: %v", userID, ok)
	if !ok {
		return 0
	}
	return userID
}

// ClearSession destroys the current session
func ClearSession(r *http.Request) error {
	return SessionManager.Destroy(r.Context())
}

// SetUserConfettiPreference stores the user's confetti preference in the session
func SetUserConfettiPreference(r *http.Request, showConfetti bool) {
	SessionManager.Put(r.Context(), "showConfetti", showConfetti)
}

// GetUserConfettiPreference retrieves the user's confetti preference from the session
func GetUserConfettiPreference(r *http.Request) bool {
	showConfetti, ok := SessionManager.Get(r.Context(), "showConfetti").(bool)
	if !ok {
		return true // Default to true if not set
	}
	return showConfetti
}

// Math problem verification helpers

// SetMathProblem stores the math problem data in the session
func SetMathProblem(r *http.Request, num1, num2, sum int) {
	SessionManager.Put(r.Context(), "mathNum1", num1)
	SessionManager.Put(r.Context(), "mathNum2", num2)
	SessionManager.Put(r.Context(), "mathSum", sum)
}

// GetMathProblem retrieves the math problem data from the session
func GetMathProblem(r *http.Request) (num1, num2, expectedSum int, ok bool) {
	num1, ok1 := SessionManager.Get(r.Context(), "mathNum1").(int)
	num2, ok2 := SessionManager.Get(r.Context(), "mathNum2").(int)
	expectedSum, ok3 := SessionManager.Get(r.Context(), "mathSum").(int)

	return num1, num2, expectedSum, ok1 && ok2 && ok3
}

// ClearMathProblem removes the math problem data from the session
func ClearMathProblem(r *http.Request) {
	SessionManager.Remove(r.Context(), "mathNum1")
	SessionManager.Remove(r.Context(), "mathNum2")
	SessionManager.Remove(r.Context(), "mathSum")
}

// GetCSRFToken returns the CSRF token from the session
func GetCSRFToken(r *http.Request) string {
	token, ok := SessionManager.Get(r.Context(), "token").(string)
	if !ok || token == "" {
		// Generate a new token if none exists
		token = GenerateCSRFToken()
		SessionManager.Put(r.Context(), "token", token)
	}
	return token
}

// GenerateCSRFToken generates a new random CSRF token
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b)
}
