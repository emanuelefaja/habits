package middleware

import (
	"database/sql"
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
	SessionManager.Lifetime = 24 * time.Hour
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
	return SessionManager.Exists(r.Context(), "userID")
}

func SetUserID(r *http.Request, userID int) {
	SessionManager.Put(r.Context(), "userID", userID)
}

func GetUserID(r *http.Request) int {
	userID, ok := SessionManager.Get(r.Context(), "userID").(int)
	if !ok {
		return 0 // or handle the error appropriately
	}
	return userID
}
