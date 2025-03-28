package web

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"mad/models"
)

// AdminDashboardHandler handles the admin dashboard page
func AdminDashboardHandler(db *sql.DB, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// AdminDownloadDBHandler handles database download for administrators
func AdminDownloadDBHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "habits.db"
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=habits.db")
		http.ServeFile(w, r, dbPath)
	}
}
