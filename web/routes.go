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

	// Setup static file handlers
	SetupStaticFileHandlers()

	// Page routes
	http.Handle("/", sessionMiddleware(HomeHandler(db, templates)))
	http.Handle("/login", sessionMiddleware(LoginHandler(db, templates)))
	http.Handle("/logout", sessionMiddleware(LogoutHandler()))
	http.Handle("/settings", sessionMiddleware(authMiddleware(SettingsHandler(db, templates))))
	http.Handle("/brand", sessionMiddleware(BrandHandler(db, templates)))
	http.Handle("/about", sessionMiddleware(AboutHandler(db, templates)))
	http.Handle("/privacy", sessionMiddleware(PrivacyHandler(db, templates)))
	http.Handle("/terms", sessionMiddleware(TermsHandler(db, templates)))
	http.Handle("/pricing", sessionMiddleware(PricingHandler(db, templates)))
	http.Handle("/tracker", sessionMiddleware(TrackerHandler(db, templates)))
	http.Handle("/masterclass", sessionMiddleware(MasterclassHandler(db, templates)))

	// New routes for module and lesson pages
	http.Handle("/masterclass/", sessionMiddleware(authMiddleware(MasterclassModuleHandler(db, templates))))
	http.Handle("/masterclass/api/", sessionMiddleware(authMiddleware(MasterclassAPIHandler(db))))

	// Admin routes
	http.Handle("/admin", sessionMiddleware(adminMiddleware(AdminDashboardHandler(db, templates))))
	http.Handle("/admin/download-db", sessionMiddleware(adminMiddleware(AdminDownloadDBHandler())))

	// Admin API routes
	http.Handle("/admin/api/user/password", sessionMiddleware(adminMiddleware(api.AdminResetPasswordHandler(db))))
	http.Handle("/admin/api/user/delete", sessionMiddleware(adminMiddleware(api.AdminDeleteUserHandler(db))))
	http.Handle("/admin/api/toggle-signups", sessionMiddleware(adminMiddleware(api.ToggleSignupStatusHandler(db))))

	// Add more routes here as you refactor them
}
