package middleware

import (
	"database/sql"
	"log"
	"mad/models"
	"net/http"
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
