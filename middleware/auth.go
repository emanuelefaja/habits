package middleware

import (
	"database/sql"
	"log"
	"mad/models"
	"net"
	"net/http"
	"strings"
)

var DB *sql.DB

func InitDB(db *sql.DB) {
	DB = db
}

func RequireGuest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsAuthenticated(r) {
			SetFlash(r, "You are already logged in")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("RequireAuth: Checking authentication for path: %s", r.URL.Path)
		if !IsAuthenticated(r) {
			log.Printf("RequireAuth: User not authenticated, redirecting to login")
			SetFlash(r, "Please log in to access this page")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		log.Printf("RequireAuth: User is authenticated, proceeding")
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("RequireAdmin: Checking authentication and admin status for path: %s", r.URL.Path)

		// First check if authenticated
		if !IsAuthenticated(r) {
			log.Printf("RequireAdmin: User not authenticated, redirecting to login")
			SetFlash(r, "Please log in to access this page")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Get user from database
		userID := GetUserID(r)
		user, err := models.GetUserByID(DB, int64(userID))
		if err != nil {
			log.Printf("RequireAdmin: Error getting user: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if user is admin
		if !user.IsAdmin {
			log.Printf("RequireAdmin: User %d attempted to access admin area", userID)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("RequireAdmin: Admin access granted for user %d", userID)
		next.ServeHTTP(w, r)
	})
}

// GetIPAddress returns the client IP address from the request
func GetIPAddress(r *http.Request) string {
	// Check for X-Forwarded-For header first (for proxies)
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		return strings.Split(ip, ",")[0]
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If there's an error splitting, just return the whole RemoteAddr
		return r.RemoteAddr
	}
	return ip
}

// GetUser returns the user from the session, or nil if not authenticated
func GetUser(r *http.Request) *models.User {
	// Get user ID from session
	userID, ok := SessionManager.Get(r.Context(), "userID").(int)
	if !ok || userID == 0 {
		return nil
	}

	// Retrieve user from DB
	user, err := models.GetUserByID(DB, int64(userID))
	if err != nil {
		return nil
	}

	return user
}
