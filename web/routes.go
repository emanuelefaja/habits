package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"mad/api"
	"mad/middleware"
	"mad/models/email"
)

// SetupRoutes configures all application routes
func SetupRoutes(db *sql.DB, templates *template.Template, emailService email.EmailService) {
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
	http.Handle("/register", sessionMiddleware(RegisterHandler(db, templates)))
	http.Handle("/settings", sessionMiddleware(authMiddleware(SettingsHandler(db, templates))))
	http.Handle("/brand", sessionMiddleware(BrandHandler(db, templates)))
	http.Handle("/about", sessionMiddleware(AboutHandler(db, templates)))
	http.Handle("/privacy", sessionMiddleware(PrivacyHandler(db, templates)))
	http.Handle("/terms", sessionMiddleware(TermsHandler(db, templates)))
	http.Handle("/pricing", sessionMiddleware(PricingHandler(db, templates)))
	http.Handle("/tracker", sessionMiddleware(TrackerHandler(db, templates)))
	http.Handle("/masterclass", sessionMiddleware(MasterclassHandler(db, templates)))
	http.Handle("/blog/", sessionMiddleware(BlogHandler(db, templates)))
	http.Handle("/changelog", sessionMiddleware(ChangelogHandler(db, templates)))
	http.Handle("/roadmap", sessionMiddleware(RoadmapHandler(db, templates)))
	http.Handle("/forgot", sessionMiddleware(ForgotPasswordHandler(db, templates)))
	http.Handle("/reset", sessionMiddleware(ResetPasswordHandler(db, templates)))

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

	// Utility routes
	http.HandleFunc("/health", HealthCheckHandler(db))

	// User API routes
	http.Handle("/api/user/profile", sessionMiddleware(authMiddleware(api.UpdateProfileHandler(db))))
	http.Handle("/api/user/password", sessionMiddleware(authMiddleware(api.UpdatePasswordHandler(db))))
	http.Handle("/api/user/delete", sessionMiddleware(authMiddleware(api.DeleteAccountHandler(db))))
	http.Handle("/api/user/export", sessionMiddleware(authMiddleware(api.ExportDataHandler(db))))
	http.Handle("/api/user/settings", sessionMiddleware(authMiddleware(api.UpdateSettingsHandler(db))))
	http.Handle("/api/user/reset-data", sessionMiddleware(authMiddleware(api.ResetDataHandler(db))))
	http.Handle("/api/user/notifications", sessionMiddleware(authMiddleware(api.UpdateNotificationPreferenceHandler(db))))
	http.Handle("/unsubscribe", sessionMiddleware(UnsubscribeHandler(db, emailService, templates)))

	// Password reset API routes
	http.Handle("/api/forgot-password", sessionMiddleware(api.ForgotPasswordHandler(db)))
	http.Handle("/api/reset-password", sessionMiddleware(api.ResetPasswordHandler(db)))

	// Roadmap API routes
	http.Handle("/api/roadmap/likes", sessionMiddleware(api.RoadmapLikesHandler(db)))
	http.Handle("/api/roadmap/ideas", sessionMiddleware(api.RoadmapIdeasHandler(db)))

	// Add more routes here as you refactor them
}
