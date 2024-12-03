package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mad/middleware"
	"mad/models"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// CreateHabitHandler handles the creation of a new habit
func CreateHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from session
		userID := middleware.GetUserID(r)
		if userID == 0 {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "User not authenticated",
			})
			return
		}

		// Decode request body
		var habit models.Habit
		if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		// Set the user ID from the session
		habit.UserID = userID

		// Check if habit already exists
		exists, err := models.HabitExists(db, habit.Name, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error checking for duplicate habit",
			})
			return
		}
		if exists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "A habit with this name already exists",
			})
			return
		}

		// Set default values
		habit.HabitType = "binary"
		habit.IsDefault = false

		// Create the habit
		if err := habit.Create(db); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error creating habit",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit created successfully",
			Data:    habit,
		})
	}
}

// GetHabitHandler retrieves a habit by ID
func GetHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		habit, err := models.GetHabitByID(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(habit)
	}
}

// UpdateHabitHandler updates an existing habit
func UpdateHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var habit models.Habit
		if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := habit.Update(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(habit)
	}
}

// DeleteHabitHandler deletes a habit by ID
func DeleteHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid habit ID", http.StatusBadRequest)
			return
		}

		habit := models.Habit{ID: id}
		if err := habit.Delete(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// CreateOrUpdateHabitLogHandler handles creating or updating a habit log
func CreateOrUpdateHabitLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse request body
		var request struct {
			HabitID int         `json:"habit_id"`
			Date    string      `json:"date"`
			Status  string      `json:"status"`
			Value   interface{} `json:"value,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		// Parse date
		date, err := time.Parse("2006-01-02", request.Date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid date format. Use YYYY-MM-DD",
			})
			return
		}

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		err = db.QueryRow("SELECT user_id FROM habits WHERE id = ?", request.HabitID).Scan(&habitUserID)
		if err != nil || habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Create habit log
		habitLog := &models.HabitLog{
			HabitID: request.HabitID,
			Date:    date,
			Status:  request.Status,
		}

		// Set value if provided
		if request.Value != nil {
			if err := habitLog.SetValue(request.Value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid value format",
				})
				return
			}
		}

		// Validate value format
		if err := habitLog.ValidateValue(db); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Create or update the log
		if err := habitLog.CreateOrUpdate(db); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error saving habit log",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Habit log saved successfully",
			Data:    habitLog,
		})
	}
}

// GetHabitLogsHandler retrieves habit logs for a date range
func GetHabitLogsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get query parameters
		habitID, err := strconv.Atoi(r.URL.Query().Get("habit_id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid habit ID",
			})
			return
		}

		startDate, err := time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid start date format. Use YYYY-MM-DD",
			})
			return
		}

		endDate, err := time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid end date format. Use YYYY-MM-DD",
			})
			return
		}

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		err = db.QueryRow("SELECT user_id FROM habits WHERE id = ?", habitID).Scan(&habitUserID)
		if err != nil || habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Get logs
		logs, err := models.GetHabitLogsByDateRange(db, habitID, startDate, endDate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habit logs",
			})
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    logs,
		})
	}
}
