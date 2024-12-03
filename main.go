package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

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
	templates := template.Must(template.New("").Funcs(template.FuncMap{
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
	}).ParseGlob("ui/*.html"))

	// Handle static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home route with session middleware and authentication check
	http.Handle("/", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Home handler: Received request for path: %s", r.URL.Path)

		if r.URL.Path != "/" {
			log.Printf("Home handler: Not root path, returning 404")
			http.NotFound(w, r)
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

		data := struct {
			User   *models.User
			Habits []models.Habit
			Flash  string
		}{
			User:   user,
			Habits: habits,
			Flash:  middleware.GetFlash(r),
		}

		err = templates.ExecuteTemplate(w, "home.html", data)
		if err != nil {
			log.Printf("Home handler: Error executing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Home handler: Successfully rendered home page")
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
		err := middleware.SessionManager.Destroy(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set flash message
		middleware.SetFlash(r, "You have been logged out successfully!")

		// Redirect to login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})))

	// Start server
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
