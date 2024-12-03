package middleware

import (
	"log"
	"net/http"
)

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
