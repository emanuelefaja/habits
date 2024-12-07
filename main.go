package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"mad/api"
	"mad/middleware"
	"mad/models"

	_ "github.com/mattn/go-sqlite3"
)

type TemplateData struct {
	Flash string
	Error string
}

func main() {
	// Initialize database
	db, err := sql.Open("sqlite3", "./habits.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize habits table
	err = models.InitializeHabitsDB(db)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize session manager
	err = middleware.InitializeSession(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create templates with custom functions
	funcMap := template.FuncMap{
		"times": func(n int) []int {
			var result []int
			for i := 0; i < n; i++ {
				result = append(result, i)
			}
			return result
		},
		"add": func(a, b int) int {
			return a + b
		},
		"dict": dict,
	}

	templates := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"ui/components/header.html",
		"ui/components/habit-modal.html",
		"ui/components/monthly-grid.html",
		"ui/components/demo-grid.html",
		"ui/components/welcome.html",
		"ui/home.html",
		"ui/settings.html",
		"ui/login.html",
		"ui/register.html",
		"ui/roadmap.html",
		"ui/habit.html",
		"ui/about.html",
		"ui/guest-home.html",
	))

	// Handle static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home route with session middleware
	http.Handle("/", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Home handler: Received request for path: %s", r.URL.Path)

		if r.URL.Path != "/" {
			log.Printf("Home handler: Not root path, returning 404")
			http.NotFound(w, r)
			return
		}

		// Check if user is authenticated
		if !middleware.IsAuthenticated(r) {
			// Render guest home page for non-authenticated users
			err := templates.ExecuteTemplate(w, "guest-home.html", nil)
			if err != nil {
				log.Printf("Home handler: Error executing guest template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// Get current user
		userID := middleware.GetUserID(r)
		log.Printf("Home handler: Got userID: %d", userID)

		user, err := models.GetUserByID(db, int64(userID))
		if err != nil {
			log.Printf("Home handler: Error getting user: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Home handler: Successfully retrieved user: %s %s", user.FirstName, user.LastName)

		// Get the habits
		habits, err := models.GetHabitsByUserID(db, userID)
		if err != nil {
			log.Printf("Home handler: Error getting habits: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// JSON encode the habits
		habitsJSON, err := json.Marshal(habits)
		if err != nil {
			log.Printf("Home handler: Error encoding habits to JSON: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

		err = templates.ExecuteTemplate(w, "home.html", data)
		if err != nil {
			log.Printf("Home handler: Error executing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Home handler: Successfully rendered home page")
	})))

	// Settings route with session middleware and authentication check
	http.Handle("/settings", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get current user
		userID := middleware.GetUserID(r)
		user, err := models.GetUserByID(db, int64(userID))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			User  *models.User
			Flash string
		}{
			User:  user,
			Flash: middleware.GetFlash(r),
		}

		err = templates.ExecuteTemplate(w, "settings.html", data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}))))

	// Register routes with session middleware
	http.Handle("/register", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			templates.ExecuteTemplate(w, "register.html", nil)
		case http.MethodPost:
			api.RegisterHandler(db, templates)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	http.Handle("/login", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			flash := middleware.GetFlash(r)
			templates.ExecuteTemplate(w, "login.html", TemplateData{Flash: flash})
		case http.MethodPost:
			api.LoginHandler(db, templates)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Logout route
	http.Handle("/logout", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Destroy the session
		err := middleware.ClearSession(r)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set flash message
		middleware.SetFlash(r, "You have been logged out successfully!")

		// Redirect to login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})))

	// Habits API routes
	http.Handle("/api/habits", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetHabitsHandler(db)(w, r)
		case http.MethodPost:
			api.CreateHabitHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	// Habit Logs API routes
	http.Handle("/api/habits/logs", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetHabitLogsHandler(db)(w, r)
		case http.MethodPost:
			api.CreateOrUpdateHabitLogHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	// Habits API routes
	http.Handle("/api/habits/bulk", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			api.BulkCreateHabitsHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	// Add new reorder route here
	http.Handle("/api/habits/reorder", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			api.UpdateHabitOrderHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	// User API routes
	http.Handle("/api/user/profile", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdateProfileHandler(db))))
	http.Handle("/api/user/password", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.UpdatePasswordHandler(db))))
	http.Handle("/api/user/delete", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.DeleteAccountHandler(db))))
	http.Handle("/api/user/export", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(api.ExportDataHandler(db))))

	// Roadmap route with session middleware but no auth requirement
	http.Handle("/roadmap", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *models.User
		var err error

		// Check if user is authenticated
		if middleware.IsAuthenticated(r) {
			userID := middleware.GetUserID(r)
			user, err = models.GetUserByID(db, int64(userID))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "roadmap",
		}

		err = templates.ExecuteTemplate(w, "roadmap.html", data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})))

	// Habit view route with session middleware and authentication check
	http.Handle("/habit/", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract habit ID from URL
		habitID, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/habit/"))
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		// Get current user
		userID := middleware.GetUserID(r)
		user, err := models.GetUserByID(db, int64(userID))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Get the habit
		habit, err := models.GetHabitByID(db, habitID)
		if err != nil {
			http.Error(w, "Habit not found", http.StatusNotFound)
			return
		}

		// Verify the habit belongs to the user
		if habit.UserID != userID {
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
			Page:  "home", // This keeps the Habits button highlighted
		}

		err = templates.ExecuteTemplate(w, "habit.html", data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}))))

	// About route with session middleware and authentication check
	http.Handle("/about", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *models.User
		var err error

		// Check if user is authenticated
		if middleware.IsAuthenticated(r) {
			userID := middleware.GetUserID(r)
			user, err = models.GetUserByID(db, int64(userID))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		data := struct {
			User *models.User
			Page string
		}{
			User: user,
			Page: "about",
		}

		err = templates.ExecuteTemplate(w, "about.html", data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})))

	// Roadmap API routes
	http.Handle("/api/roadmap/likes", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetRoadmapLikesHandler(db)(w, r)
		case http.MethodPost:
			api.ToggleRoadmapLikeHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Roadmap API routes
	http.Handle("/api/roadmap/ideas", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.SubmitRoadmapIdeaHandler(db)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Start server
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}
