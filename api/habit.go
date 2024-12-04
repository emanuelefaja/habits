package api

import (
	"database/sql"
	"encoding/json"
	"log"
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

// CreateHabitRequest represents a request to create a new habit
type CreateHabitRequest struct {
	Name      string           `json:"name"`
	HabitType models.HabitType `json:"habit_type"`
}

// BulkHabitRequest represents a request to create multiple habits
type BulkHabitRequest struct {
	Name      string           `json:"name"`
	HabitType models.HabitType `json:"habit_type"`
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
		var request CreateHabitRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		// Validate habit type
		if request.HabitType != models.BinaryHabit && request.HabitType != models.NumericHabit {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid habit type. Must be 'binary' or 'numeric'",
			})
			return
		}

		habit := models.Habit{
			UserID:    userID,
			Name:      request.Name,
			HabitType: request.HabitType,
			IsDefault: false,
		}

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
			Status  string      `json:"status,omitempty"` // Optional for numeric habits
			Value   interface{} `json:"value,omitempty"`  // Required for numeric habits
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

		// Get habit type first
		var habitType models.HabitType
		err = db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", request.HabitID).Scan(&habitType)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habit type",
			})
			return
		}

		// Validate input based on habit type
		habitLog := &models.HabitLog{
			HabitID: request.HabitID,
			Date:    date,
		}

		if habitType == models.BinaryHabit {
			if request.Status == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Status is required for binary habits",
				})
				return
			}
			habitLog.Status = request.Status
		} else {
			// For numeric habits
			if request.Value == nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Value is required for numeric habits",
				})
				return
			}
			// Use the provided status or default to "done"
			if request.Status != "" {
				habitLog.Status = request.Status
			} else {
				habitLog.Status = "done"
			}
			if err := habitLog.SetValue(request.Value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Invalid numeric value",
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

// BulkCreateHabitsHandler handles creating multiple habits at once
func BulkCreateHabitsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("BulkCreateHabitsHandler: Method %s", r.Method)
		userID := middleware.GetUserID(r)
		log.Printf("BulkCreateHabitsHandler: UserID %d", userID)

		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var habits []BulkHabitRequest
		if err := json.NewDecoder(r.Body).Decode(&habits); err != nil {
			log.Printf("BulkCreateHabitsHandler: Error decoding request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("BulkCreateHabitsHandler: Received habits: %+v", habits)

		for _, habit := range habits {
			exists, err := models.HabitExists(db, habit.Name, userID)
			if err != nil {
				log.Printf("BulkCreateHabitsHandler: Error checking habit existence: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if exists {
				log.Printf("BulkCreateHabitsHandler: Habit already exists: %s", habit.Name)
				continue
			}

			newHabit := &models.Habit{
				UserID:    userID,
				Name:      habit.Name,
				HabitType: habit.HabitType,
				IsDefault: false,
			}
			if err := newHabit.Create(db); err != nil {
				log.Printf("BulkCreateHabitsHandler: Error creating habit: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("BulkCreateHabitsHandler: Created habit: %s", habit.Name)
		}

		log.Printf("BulkCreateHabitsHandler: Successfully created all habits")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

// GetHabitsHandler retrieves all habits for a user
func GetHabitsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID := middleware.GetUserID(r)
		if userID == 0 {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "User not authenticated",
			})
			return
		}

		habits, err := models.GetHabitsByUserID(db, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error retrieving habits",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    habits,
		})
	}
}
