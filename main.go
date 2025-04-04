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
	"mad/web"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// Helper variables
var (
	// Rate limiters
	webUnsubscribeLimiter = middleware.NewRateLimiter(10, time.Hour) // 10 attempts per hour
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

	// Initialize campaign manager
	if emailService != nil {
		campaignManager := email.NewCampaignManager(db, emailService)

		// Set the campaign manager in the email service if it's the SMTP implementation
		if smtpService, ok := emailService.(*email.SMTPEmailService); ok {
			smtpService.SetCampaignManager(campaignManager)
		}
	}

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

	// NEW: Set up the home route
	web.SetupRoutes(db, templates)

	// Routes

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

		data := web.TemplateData{
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
		if r.URL.Path != "/courses/digital-detox" {
			http.NotFound(w, r)
			return
		}

		// Create random numbers for math verification
		num1 := rand.Intn(20) + 1 // Random number between 1-20
		num2 := rand.Intn(20) + 1 // Random number between 1-20

		// Default data map
		data := map[string]interface{}{
			"IsAuthenticated": middleware.IsAuthenticated(r),
			"IsSubscribed":    false,
			"UserEmail":       "",
			"UserFirstName":   "",
			"CSRFToken":       middleware.GetCSRFToken(r),
			"MathNum1":        num1,
			"MathNum2":        num2,
		}

		// Add user data if logged in
		if data["IsAuthenticated"].(bool) {
			user, err := getAuthenticatedUser(r, db)
			if err == nil && user != nil {
				data["User"] = user
				data["UserEmail"] = user.Email
				data["UserFirstName"] = user.FirstName

				// Check if the user is already subscribed to the Digital Detox campaign
				svc, ok := emailService.(email.EmailService)
				if ok && svc != nil {
					campaignManager := svc.GetCampaignManager()
					if campaignManager != nil {
						subscriptions, err := campaignManager.GetUserSubscriptions(int(user.ID))
						if err == nil {
							// Check if user is subscribed to the Digital Detox campaign
							for _, subscription := range subscriptions {
								if subscription.CampaignID == "digital-detox" && subscription.Status == "active" {
									data["IsSubscribed"] = true
									break
								}
							}
						}
					}
				}
			}
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

	// Roadmap API handlers
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

	// Campaign API handlers
	http.Handle("/api/campaigns/subscribe", middleware.SessionManager.LoadAndSave(http.HandlerFunc(api.SubscribeToCampaign)))
	http.Handle("/api/campaigns/unsubscribe", middleware.SessionManager.LoadAndSave(http.HandlerFunc(api.UnsubscribeFromCampaign)))
	http.Handle("/api/campaigns/subscriptions", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(api.GetSubscriptions))))
	http.Handle("/api/campaigns/preferences", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(api.UpdateSubscriptionPreferences))))

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

	// Unsubscribe handler
	http.Handle("/unsubscribe", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the campaign manager from the email service
		campaignManager := emailService.GetCampaignManager()
		if campaignManager == nil {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
			return
		}

		// For GET requests, just show the unsubscribe page
		if r.Method == http.MethodGet {
			userEmail := r.URL.Query().Get("email")
			campaignID := r.URL.Query().Get("campaign")
			token := r.URL.Query().Get("token")

			log.Printf("Unsubscribe GET request: email=%s, campaign=%s, token=%s", userEmail, campaignID, token)

			if userEmail == "" || campaignID == "" {
				log.Printf("Missing query parameters: email=%s, campaign=%s", userEmail, campaignID)
				http.Error(w, "Missing required parameters", http.StatusBadRequest)
				return
			}

			// Get campaign details for display
			campaign, err := email.GetCampaign(campaignID)
			if err != nil {
				log.Printf("Error getting campaign: %v", err)
				http.Error(w, "Invalid campaign", http.StatusNotFound)
				return
			}

			data := struct {
				Email         string
				CampaignID    string
				CampaignName  string
				CampaignEmoji string
				Token         string
				Quote         struct {
					Text   string
					Author string
				}
				Unsubscribed bool
			}{
				Email:         userEmail,
				CampaignID:    campaignID,
				CampaignName:  campaign.Name,
				CampaignEmoji: campaign.Emoji,
				Token:         token,
				Quote: struct {
					Text   string
					Author string
				}{
					Text:   "Small habits make big changes.",
					Author: "The Habits Company",
				},
				Unsubscribed: false,
			}
			renderTemplate(w, templates, "unsubscribe.html", data)
			return
		}

		// For POST requests, handle the unsubscribe action
		if r.Method == http.MethodPost {
			// Parse form data
			if err := r.ParseForm(); err != nil {
				log.Printf("Error parsing form data: %v", err)
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}

			// Get values from form data instead of URL query parameters
			formEmail := r.PostFormValue("email")
			formCampaignID := r.PostFormValue("campaign_id")
			formToken := r.PostFormValue("token")

			log.Printf("Unsubscribe POST request: email=%s, campaign=%s, token=%s", formEmail, formCampaignID, formToken)

			if formEmail == "" || formCampaignID == "" || formToken == "" {
				log.Printf("Missing form parameters: email=%s, campaign=%s, token=%s", formEmail, formCampaignID, formToken)
				http.Error(w, "Missing required parameters", http.StatusBadRequest)
				return
			}

			// Apply rate limiting - 10 attempts per hour per IP
			remaining, resetTime, err := webUnsubscribeLimiter.CheckLimit(r)
			if err != nil {
				log.Printf("Rate limit check error: %v", err)
				http.Error(w, "Error processing request", http.StatusInternalServerError)
				return
			}
			if remaining == 0 {
				waitDuration := time.Until(resetTime)
				http.Error(w, fmt.Sprintf("Too many unsubscribe attempts. Please try again in %d minutes.", int(waitDuration.Minutes())+1), http.StatusTooManyRequests)
				return
			}

			// With token validation:
			valid, err := campaignManager.ValidateUnsubscribeToken(formEmail, formCampaignID, formToken)
			if err != nil {
				log.Printf("Error validating token: %v", err)
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}

			if !valid {
				log.Printf("Invalid token for email=%s, campaign=%s", formEmail, formCampaignID)
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}

			// Get campaign details for the response
			campaign, err := email.GetCampaign(formCampaignID)
			if err != nil {
				log.Printf("Error getting campaign: %v", err)
				http.Error(w, "Invalid campaign", http.StatusNotFound)
				return
			}

			err = campaignManager.UnsubscribeUser(formEmail, formCampaignID)
			if err != nil {
				log.Printf("Error unsubscribing user: %v", err)
				http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
				return
			}

			log.Printf("Successfully unsubscribed %s from campaign %s", formEmail, formCampaignID)

			data := struct {
				Success       bool
				Email         string
				CampaignID    string
				CampaignName  string
				CampaignEmoji string
				Quote         struct {
					Text   string
					Author string
				}
				Unsubscribed bool
				Token        string
			}{
				Success:       true,
				Email:         formEmail,
				CampaignID:    formCampaignID,
				CampaignName:  campaign.Name,
				CampaignEmoji: campaign.Emoji,
				Quote: struct {
					Text   string
					Author string
				}{
					Text:   "Small habits make big changes.",
					Author: "The Habits Company",
				},
				Unsubscribed: true,
				Token:        formToken,
			}
			renderTemplate(w, templates, "unsubscribe.html", data)
			return
		}

		handleNotAllowed(w, http.MethodGet, http.MethodPost)
	})))

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
