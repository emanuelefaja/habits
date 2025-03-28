package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"mad/api"
	"mad/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes(db *sql.DB, templates *template.Template) {
	// Common middleware
	sessionMiddleware := middleware.SessionManager.LoadAndSave
	authMiddleware := middleware.RequireAuth
	adminMiddleware := middleware.RequireAdmin

	// Page routes
	http.Handle("/", sessionMiddleware(HomeHandler(db, templates)))
	http.Handle("/login", sessionMiddleware(LoginHandler(db, templates)))
	http.Handle("/logout", sessionMiddleware(LogoutHandler()))
	http.Handle("/settings", sessionMiddleware(authMiddleware(SettingsHandler(db, templates))))

	// Admin routes
	http.Handle("/admin", sessionMiddleware(adminMiddleware(AdminDashboardHandler(db, templates))))
	http.Handle("/admin/download-db", sessionMiddleware(adminMiddleware(AdminDownloadDBHandler())))

	// Admin API routes
	http.Handle("/admin/api/user/password", sessionMiddleware(adminMiddleware(api.AdminResetPasswordHandler(db))))
	http.Handle("/admin/api/user/delete", sessionMiddleware(adminMiddleware(api.AdminDeleteUserHandler(db))))
	http.Handle("/admin/api/toggle-signups", sessionMiddleware(adminMiddleware(api.ToggleSignupStatusHandler(db))))

	// Add more routes here as you refactor them
}
