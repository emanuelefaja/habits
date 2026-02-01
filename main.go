package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mad/api"
	"mad/middleware"
	"mad/models"
	"mad/models/email"
	"mad/web"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using default environment variables")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./habits.db"
	}

	// Open and verify DB connection
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize DB and services
	if err := models.InitDB(db); err != nil {
		log.Fatal("Error initializing database:", err)
	}

	// Run database migrations
	if err := models.MigrateDB(db); err != nil {
		log.Fatal("Error migrating database:", err)
	}

	if err := models.InitializeHabitsDB(db); err != nil {
		log.Fatal(err)
	}
	if err := middleware.InitializeSession(db); err != nil {
		log.Fatal(err)
	}

	// Initialize email service
	emailService, err := email.NewSMTPEmailService(email.SMTPConfig{
		Host:        os.Getenv("SMTP_HOST"),
		Port:        587, // Default SMTP port
		Username:    os.Getenv("SMTP_USERNAME"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		FromName:    os.Getenv("SMTP_FROM_NAME"),
		FromEmail:   os.Getenv("SMTP_FROM_EMAIL"),
		TemplateDir: "./ui/email",
		Secure:      true,
		RequireTLS:  true,
	})
	if err != nil {
		log.Printf("Warning: Could not initialize email service: %v", err)
	}

	// Pass the email service to the API handlers
	api.InitEmailService(emailService)

	// Initialize and start the scheduler for email notifications
	scheduler := models.NewScheduler(db, emailService)
	if err := scheduler.Start(); err != nil {
		log.Printf("Warning: Could not start email scheduler: %v", err)
	} else {
		log.Println("Email notification scheduler started successfully")
	}

	// Only seed users if in development environment
	if os.Getenv("APP_ENV") == "development" {
		err = models.SeedUsers(db)
		if err != nil {
			log.Println("Warning: Could not seed users:", err)
		}
	}

	api.StartGitHubSync(db)
	middleware.InitDB(db)

	// Load templates from the web package
	templates, err := web.LoadTemplates()
	if err != nil {
		log.Fatalf("Template parsing error: %v", err)
	}

	// Set up routes
	web.SetupRoutes(db, templates, emailService)

	// Routes
	// Habits API
	http.Handle("/api/habits", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetHabitsHandler(db)(w, r)
		case http.MethodPost:
			api.CreateHabitHandler(db)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}))))

	http.Handle("/api/habits/logs", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetHabitLogsHandler(db)(w, r)
		case http.MethodPost:
			api.CreateOrUpdateHabitLogHandler(db)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}))))

	http.Handle("/api/habits/bulk", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.BulkCreateHabitsHandler(db)(w, r)
		} else {
			handleNotAllowed(w, http.MethodPost)
		}
	}))))

	http.Handle("/api/habits/reorder", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.UpdateHabitOrderHandler(db)(w, r)
		} else {
			handleNotAllowed(w, http.MethodPost)
		}
	}))))

	// Habit View
	http.Handle("/habit/", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		habitID, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/habit/"))
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		habit, err := models.GetHabitByID(db, habitID)
		if err != nil {
			http.Error(w, "Habit not found", http.StatusNotFound)
			return
		}
		if habit.UserID != middleware.GetUserID(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		data := struct {
			User  *models.User
			Habit *models.Habit
			Page  string
		}{
			User:  user,
			Habit: habit,
			Page:  "home",
		}
		renderTemplate(w, templates, "habit.html", data)
	}))))

	// Habit Logs Deletion
	http.Handle("/api/habits/logs/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			handleNotAllowed(w, http.MethodDelete)
			return
		}
		api.DeleteHabitLogHandler(db)(w, r)
	}))))

	// Habit Deletion
	http.Handle("/api/habits/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			handleNotAllowed(w, http.MethodDelete)
			return
		}
		api.DeleteHabitHandler(db)(w, r)
	}))))

	http.Handle("/api/habits/stats", middleware.SessionManager.LoadAndSave(http.HandlerFunc(api.HandleGetHabitStats(db))))

	// Habit Name Update
	http.Handle("/api/habits/update-name", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handleNotAllowed(w, http.MethodPost)
			return
		}
		api.UpdateHabitNameHandler(db)(w, r)
	}))))

	// Commits API
	http.Handle("/api/commits", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commits, err := models.GetCommits(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, commits)
	})))

	http.Handle("/goals", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "goals",
		}
		renderTemplate(w, templates, "goals.html", data)
	}))))

	// Goals API
	http.Handle("/api/goals", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetGoalsHandler(db)(w, r)
		case http.MethodPost:
			api.CreateGoalHandler(db)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	}))))

	http.Handle("/api/goals/reorder", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			handleNotAllowed(w, http.MethodPut)
			return
		}
		api.ReorderGoalsHandler(db)(w, r)
	}))))

	http.Handle("/api/goals/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			handleNotAllowed(w, http.MethodDelete)
			return
		}
		api.DeleteGoalHandler(db)(w, r)
	}))))

	http.Handle("/api/goals/update", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			handleNotAllowed(w, http.MethodPut)
			return
		}
		api.UpdateGoalHandler(db)(w, r)
	}))))

	// Unsubscribe handler - Now moved to web/unsubscribe_handler.go

	// Changelog route is now in web/routes.go

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Helper functions (This can be deleted once full main.go refactor is complete)

func renderTemplate(w http.ResponseWriter, templates *template.Template, name string, data interface{}) {
	// Use a buffer to render the template first
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log the size of the response
	responseSize := buf.Len()
	log.Printf("Template %s rendered with size: %d bytes", name, responseSize)

	// Then write the buffered content to the response writer
	_, err := buf.WriteTo(w)
	if err != nil {
		// Check if the error is due to a client disconnection
		if strings.Contains(err.Error(), "write: broken pipe") ||
			strings.Contains(err.Error(), "client disconnected") ||
			strings.Contains(err.Error(), "connection reset by peer") {
			log.Printf("Client disconnected while sending template %s: %v", name, err)
			return // Don't try to write an error response to a disconnected client
		}

		log.Printf("Error writing template %s to response: %v", name, err)
		// At this point, we may not be able to write an error response
		// since we've already started writing the response
	}
}

func respondJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func handleNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func getAuthenticatedUser(r *http.Request, db *sql.DB) (*models.User, error) {
	if !middleware.IsAuthenticated(r) {
		return nil, nil
	}
	userID := middleware.GetUserID(r)
	return models.GetUserByID(db, int64(userID))
}
