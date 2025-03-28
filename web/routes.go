package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"mad/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes(db *sql.DB, templates *template.Template) {
	// Common middleware
	sessionMiddleware := middleware.SessionManager.LoadAndSave
	authMiddleware := middleware.RequireAuth

	// Page routes
	http.Handle("/", sessionMiddleware(HomeHandler(db, templates)))
	http.Handle("/login", sessionMiddleware(LoginHandler(db, templates)))
	http.Handle("/logout", sessionMiddleware(LogoutHandler()))
	http.Handle("/settings", sessionMiddleware(authMiddleware(SettingsHandler(db, templates))))

	// Add more routes here as you refactor them
}
