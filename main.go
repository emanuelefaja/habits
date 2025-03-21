package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mad/api"
	"mad/middleware"
	"mad/models"
	"mad/models/email"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type TemplateData struct {
	Flash      string
	Error      string
	Token      string
	IsLoggedIn bool
	Email      string
}

// timeoutResponseWriter is a custom ResponseWriter that adds timeout functionality
type timeoutResponseWriter struct {
	http.ResponseWriter
	timeout time.Duration
	start   time.Time
}

func newTimeoutResponseWriter(w http.ResponseWriter, timeout time.Duration) *timeoutResponseWriter {
	return &timeoutResponseWriter{
		ResponseWriter: w,
		timeout:        timeout,
		start:          time.Now(),
	}
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	// Check if we've exceeded the timeout
	if time.Since(w.start) > w.timeout {
		return 0, fmt.Errorf("response timeout exceeded")
	}
	return w.ResponseWriter.Write(b)
}

func (w *timeoutResponseWriter) WriteHeader(statusCode int) {
	// Check if we've exceeded the timeout
	if time.Since(w.start) > w.timeout {
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func main() {
	// Initialize random seed for Go versions before 1.20
	// In Go 1.20+ this is no longer needed as it's done automatically
	// This is kept for backward compatibility
	rand.Seed(time.Now().UnixNano())

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

	// Template functions
	funcMap := template.FuncMap{
		"times": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i
			}
			return result
		},
		"add": func(a, b int) int {
			return a + b
		},
		"dict": dict,
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}

	templates := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"ui/components/header.html",
		"ui/components/habit-modal.html",
		"ui/components/monthly-grid.html",
		"ui/components/demo-grid.html",
		"ui/components/welcome.html",
		"ui/components/yearly-grid.html",
		"ui/components/head.html",
		"ui/components/footer.html",
		"ui/components/sum-line-graph.html",
		"ui/components/goal.html",
		"ui/home.html",
		"ui/settings.html",
		"ui/login.html",
		"ui/register.html",
		"ui/roadmap.html",
		"ui/habits/habit.html",
		"ui/habits/binary.html",
		"ui/habits/numeric.html",
		"ui/habits/choice.html",
		"ui/habits/set-rep.html",
		"ui/about.html",
		"ui/guest-home.html",
		"ui/admin.html",
		"ui/changelog.html",
		"ui/blog/blog.html",
		"ui/blog/post.html",
		"ui/goals.html",
		"ui/forgot.html",
		"ui/reset.html",
		"ui/courses/digital-detox.html",
	))

	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir("static/icons"))))
	http.Handle("/content/media/", http.StripPrefix("/content/media/", http.FileServer(http.Dir("content/media"))))

	// Manifest and Service Worker
	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFileWithContentType(w, r, "static/manifest.json", "application/manifest+json")
	})
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Service-Worker-Allowed", "/")
		http.ServeFile(w, r, "static/sw.js")
	})

	// Sitemap
	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFileWithContentType(w, r, "static/sitemap.xml", "application/xml")
	})

	// Routes
	http.Handle("/", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer with a timeout
		tw := newTimeoutResponseWriter(w, 10*time.Second)

		if r.URL.Path != "/" {
			http.NotFound(tw, r)
			return
		}

		if !middleware.IsAuthenticated(r) {
			// Guest
			if err := templates.ExecuteTemplate(tw, "guest-home.html", nil); err != nil {
				// Check if the error is due to a client disconnection
				if strings.Contains(err.Error(), "write: broken pipe") ||
					strings.Contains(err.Error(), "client disconnected") ||
					strings.Contains(err.Error(), "connection reset by peer") ||
					strings.Contains(err.Error(), "response timeout exceeded") {
					log.Printf("Client disconnected while rendering guest-home.html: %v", err)
					return
				}
				http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			log.Printf("Error getting authenticated user: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only get necessary habit data for the current view
		habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
		if err != nil {
			log.Printf("Error getting habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Limit the amount of data sent to the template
		habitsJSON, err := json.Marshal(habits)
		if err != nil {
			log.Printf("Error marshaling habits: %v", err)
			http.Error(tw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			User       *models.User
			HabitsJSON template.JS
			Flash      string
		}{
			User:       user,
			HabitsJSON: template.JS(habitsJSON),
			Flash:      middleware.GetFlash(r),
		}
		renderTemplate(tw, templates, "home.html", data)
	})))

	http.Handle("/settings", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Debug: Print user settings
		log.Printf("User settings: confetti=%v, weekdays=%v, notifications=%v", user.ShowConfetti, user.ShowWeekdays, user.NotificationEnabled)

		data := struct {
			User  *models.User
			Flash string
		}{
			User:  user,
			Flash: middleware.GetFlash(r),
		}
		renderTemplate(w, templates, "settings.html", data)
	}))))

	// Authentication Routes
	http.Handle("/register", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Check if signups are allowed
			allowSignups, err := models.GetSignupStatus(db)
			if err != nil {
				log.Printf("Error checking signup status: %v", err)
				// Default to allowing signups if there's an error
			} else if !allowSignups {
				// Redirect to login page with a message
				middleware.SetFlash(r, "Registration is currently disabled ❌")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Generate math problem for human verification
			num1 := rand.Intn(20) + 1 // Random number between 1-20
			num2 := rand.Intn(20) + 1 // Random number between 1-20
			sum := num1 + num2

			// Store in session
			middleware.SetMathProblem(r, num1, num2, sum)

			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			// Pass to template
			data := map[string]interface{}{
				"MathNum1": num1,
				"MathNum2": num2,
				"Quote":    quote,
			}

			renderTemplate(w, templates, "register.html", data)
		case http.MethodPost:
			api.RegisterHandler(db, templates)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})))

	http.Handle("/login", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			data := TemplateData{
				Flash: middleware.GetFlash(r),
			}

			// Add quote to the template data
			templateData := map[string]interface{}{
				"Flash": data.Flash,
				"Error": data.Error,
				"Quote": quote,
			}

			renderTemplate(w, templates, "login.html", templateData)
		case http.MethodPost:
			api.LoginHandler(db, templates)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})))

	http.Handle("/forgot", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handleNotAllowed(w, http.MethodGet)
			return
		}

		// Get a random quote
		quote, err := models.GetRandomQuote()
		if err != nil {
			log.Printf("Error getting random quote: %v", err)
			// Continue with default quote from the function
		}

		data := TemplateData{
			IsLoggedIn: middleware.IsAuthenticated(r),
		}
		if data.IsLoggedIn {
			user, err := getAuthenticatedUser(r, db)
			if err == nil {
				data.Email = user.Email
			}
		}

		// Add quote to the template data
		templateData := map[string]interface{}{
			"IsLoggedIn": data.IsLoggedIn,
			"Email":      data.Email,
			"Quote":      quote,
		}

		renderTemplate(w, templates, "forgot.html", templateData)
	})))

	http.Handle("/logout", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handleNotAllowed(w, http.MethodPost)
			return
		}
		if err := middleware.ClearSession(r); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		middleware.SetFlash(r, "You have been logged out successfully!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})))

	// Password Reset Routes
	http.Handle("/reset", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Redirect(w, r, "/forgot", http.StatusSeeOther)
				return
			}

			// Get a random quote
			quote, err := models.GetRandomQuote()
			if err != nil {
				log.Printf("Error getting random quote: %v", err)
				// Continue with default quote from the function
			}

			// Add quote to the template data
			templateData := map[string]interface{}{
				"Token": token,
				"Flash": middleware.GetFlash(r),
				"Quote": quote,
			}

			renderTemplate(w, templates, "reset.html", templateData)
		default:
			handleNotAllowed(w, http.MethodGet)
		}
	})))

	http.Handle("/api/forgot-password", middleware.SessionManager.LoadAndSave(http.HandlerFunc(api.ForgotPasswordHandler(db))))
	http.Handle("/api/reset-password", middleware.SessionManager.LoadAndSave(http.HandlerFunc(api.ResetPasswordHandler(db))))

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

	// User API
	http.Handle("/api/user/profile", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdateProfileHandler(db))))
	http.Handle("/api/user/password", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdatePasswordHandler(db))))
	http.Handle("/api/user/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.DeleteAccountHandler(db))))
	http.Handle("/api/user/export", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.ExportDataHandler(db))))
	http.Handle("/api/user/settings", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdateSettingsHandler(db))))
	http.Handle("/api/user/reset-data", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.ResetDataHandler(db))))
	http.Handle("/api/user/notifications", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdateNotificationPreferenceHandler(db))))

	// Roadmap (no auth required, but session loaded)
	http.Handle("/roadmap", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "roadmap",
		}
		renderTemplate(w, templates, "roadmap.html", data)
	})))

	// Digital Detox Course
	http.Handle("/courses/digital-detox", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "courses",
		}
		renderTemplate(w, templates, "digital-detox.html", data)
	})))

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

	// About
	http.Handle("/about", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "about",
		}
		renderTemplate(w, templates, "about.html", data)
	})))

	// Roadmap API
	http.Handle("/api/roadmap/likes", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetRoadmapLikesHandler(db)(w, r)
		case http.MethodPost:
			api.ToggleRoadmapLikeHandler(db)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})))

	http.Handle("/api/roadmap/ideas", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.SubmitRoadmapIdeaHandler(db)(w, r)
		} else {
			handleNotAllowed(w, http.MethodPost)
		}
	})))

	// Admin
	http.Handle("/admin", middleware.SessionManager.LoadAndSave(
		middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := getAuthenticatedUser(r, db)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			totalUsers, err := models.GetTotalUsers(db)
			if err != nil {
				log.Printf("Error getting total users: %v", err)
				totalUsers = 0
			}

			totalHabits, err := models.GetTotalHabits(db)
			if err != nil {
				log.Printf("Error getting total habits: %v", err)
				totalHabits = 0
			}

			totalHabitLogs, err := models.GetTotalHabitLogs(db)
			if err != nil {
				log.Printf("Error getting total habit logs: %v", err)
				totalHabitLogs = 0
			}

			totalGoals, err := models.GetTotalGoals(db)
			if err != nil {
				log.Printf("Error getting total goals: %v", err)
				totalGoals = 0
			}

			users, err := models.GetAllUsers(db)
			if err != nil {
				log.Printf("Error getting all users: %v", err)
				users = []*models.User{}
			}

			// Get signup status
			allowSignups, err := models.GetSignupStatus(db)
			if err != nil {
				log.Printf("Error getting signup status: %v", err)
				allowSignups = true // Default to allowing signups
			}

			data := struct {
				User           *models.User
				Users          []*models.User
				TotalUsers     int
				TotalHabits    int
				TotalHabitLogs int
				TotalGoals     int
				AllowSignups   bool
			}{
				User:           user,
				Users:          users,
				TotalUsers:     totalUsers,
				TotalHabits:    totalHabits,
				TotalHabitLogs: totalHabitLogs,
				TotalGoals:     totalGoals,
				AllowSignups:   allowSignups,
			}

			renderTemplate(w, templates, "admin.html", data)
		}))))

	// Admin APIs
	http.Handle("/admin/api/user/password", middleware.SessionManager.LoadAndSave(middleware.RequireAdmin(api.AdminResetPasswordHandler(db))))
	http.Handle("/admin/api/user/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAdmin(api.AdminDeleteUserHandler(db))))
	http.Handle("/admin/api/toggle-signups", middleware.SessionManager.LoadAndSave(middleware.RequireAdmin(api.ToggleSignupStatusHandler(db))))

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

	// Changelog
	http.Handle("/changelog", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := getAuthenticatedUser(r, db)
		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "changelog",
		}
		renderTemplate(w, templates, "changelog.html", data)
	})))

	// Commits API
	http.Handle("/api/commits", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commits, err := models.GetCommits(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, commits)
	})))

	// Health Check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			respondJSON(w, map[string]string{"status": "error", "message": "Database connection failed"})
			return
		}
		respondJSON(w, map[string]string{"status": "healthy"})
	})

	// Blog
	http.Handle("/blog/", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/blog")
		blogService := models.GetBlogService()

		user, _ := getAuthenticatedUser(r, db)

		if path == "" || path == "/" {
			posts := blogService.GetAllPosts()
			data := struct {
				User  *models.User
				Posts []*models.BlogPost
				Page  string
			}{
				User:  user,
				Posts: posts,
				Page:  "blog",
			}
			renderTemplate(w, templates, "blog.html", data)
			return
		}

		slug := strings.TrimPrefix(path, "/")
		post, exists := blogService.GetPost(slug)
		if !exists {
			http.NotFound(w, r)
			return
		}

		data := struct {
			User *models.User
			Post *models.BlogPost
			Page string
		}{
			User: user,
			Post: post,
			Page: "blog",
		}
		renderTemplate(w, templates, "post.html", data)
	})))

	// http.Handle("/books", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	user, _ := getAuthenticatedUser(r, db)
	// 	data := struct {
	// 		User *models.User
	// 		Page string
	// 	}{
	// 		User: user,
	// 		Page: "books",
	// 	}
	// 	renderTemplate(w, templates, "books.html", data)
	// })))

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

	// // Profile
	// http.Handle("/profile", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	user, _ := getAuthenticatedUser(r, db)
	// 	data := struct {
	// 		User *models.User
	// 	}{
	// 		User: user,
	// 	}
	// 	renderTemplate(w, templates, "profile.html", data)
	// })))

	// Admin Download DB
	http.Handle("/admin/download-db", middleware.SessionManager.LoadAndSave(middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "habits.db"
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=habits.db")
		http.ServeFile(w, r, dbPath)
	}))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Helper functions

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	d := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		d[key] = values[i+1]
	}
	return d, nil
}

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

func serveStaticFileWithContentType(w http.ResponseWriter, r *http.Request, filePath, contentType string) {
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}
