package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"mad/api"
	"mad/middleware"
	"mad/models"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type TemplateData struct {
	Flash string
	Error string
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./habits.db"
	}

	// Open and verify DB connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Initialize DB and services
	if err := models.InitDB(db); err != nil {
		log.Fatal("Error initializing database:", err)
	}
	if err := models.InitializeHabitsDB(db); err != nil {
		log.Fatal(err)
	}
	if err := middleware.InitializeSession(db); err != nil {
		log.Fatal(err)
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
	))

	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir("static/icons"))))

	// Manifest and Service Worker
	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFileWithContentType(w, r, "static/manifest.json", "application/manifest+json")
	})
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Service-Worker-Allowed", "/")
		http.ServeFile(w, r, "static/sw.js")
	})

	// Routes
	http.Handle("/", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if !middleware.IsAuthenticated(r) {
			// Guest
			if err := templates.ExecuteTemplate(w, "guest-home.html", nil); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		habits, err := models.GetHabitsByUserID(db, middleware.GetUserID(r))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		habitsJSON, err := json.Marshal(habits)
		if err != nil {
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
		renderTemplate(w, templates, "home.html", data)
	})))

	http.Handle("/settings", middleware.SessionManager.LoadAndSave(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := getAuthenticatedUser(r, db)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Debug: Print user settings
		log.Printf("User settings: confetti=%v, weekdays=%v", user.ShowConfetti, user.ShowWeekdays)

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
			renderTemplate(w, templates, "register.html", nil)
		case http.MethodPost:
			api.RegisterHandler(db, templates)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})))

	http.Handle("/login", middleware.SessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			renderTemplate(w, templates, "login.html", TemplateData{Flash: middleware.GetFlash(r)})
		case http.MethodPost:
			api.LoginHandler(db, templates)(w, r)
		default:
			handleNotAllowed(w, http.MethodGet, http.MethodPost)
		}
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
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			data := struct {
				User           *models.User
				Users          []*models.User
				TotalUsers     int
				TotalHabits    int
				TotalHabitLogs int
				TotalGoals     int
				Page           string
			}{
				User:           user,
				Users:          users,
				TotalUsers:     totalUsers,
				TotalHabits:    totalHabits,
				TotalHabitLogs: totalHabitLogs,
				TotalGoals:     totalGoals,
				Page:           "admin",
			}
			renderTemplate(w, templates, "admin.html", data)
		}))))

	// Admin APIs
	http.Handle("/admin/api/user/password", middleware.SessionManager.LoadAndSave(middleware.RequireAdmin(api.AdminResetPasswordHandler(db))))

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
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
