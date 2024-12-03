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

	// Initialize users table
	err = models.InitializeDB(db)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize session manager
	err = middleware.InitializeSession(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create templates
	templates := template.Must(template.ParseGlob("ui/*.html"))

	// Handle static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

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

	// Start server
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
