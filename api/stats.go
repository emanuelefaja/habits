package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"mad/middleware"
	"mad/models"
)

// HandleGetHabitStats returns statistics for a habit
func HandleGetHabitStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set content type header first
		w.Header().Set("Content-Type", "application/json")

		// Create a response function to ensure consistent response format
		sendResponse := func(status int, success bool, message string, data interface{}) {
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(APIResponse{
				Success: success,
				Message: message,
				Data:    data,
			})
		}

		// Get habit ID from URL query parameter
		idStr := r.URL.Query().Get("id")
		habitID, err := strconv.Atoi(idStr)
		if err != nil {
			sendResponse(http.StatusBadRequest, false, "Invalid habit ID", nil)
			return
		}

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		err = db.QueryRow("SELECT user_id FROM habits WHERE id = ?", habitID).Scan(&habitUserID)
		if err != nil {
			log.Printf("Error getting habit user ID: %v", err)
			sendResponse(http.StatusInternalServerError, false, "Error getting habit", nil)
			return
		}

		if habitUserID != userID {
			sendResponse(http.StatusForbidden, false, "Unauthorized access to habit", nil)
			return
		}

		// Get habit type
		var habitType models.HabitType
		err = db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", habitID).Scan(&habitType)
		if err != nil {
			log.Printf("Error getting habit type: %v", err)
			sendResponse(http.StatusInternalServerError, false, "Error getting habit type", nil)
			return
		}

		var stats interface{}
		switch habitType {
		case models.BinaryHabit:
			stats, err = models.GetBinaryHabitStats(db, habitID)
		case models.NumericHabit:
			stats, err = models.GetNumericHabitStats(db, habitID)
		case models.OptionSelectHabit:
			stats, err = models.GetChoiceHabitStats(db, habitID)
		case models.SetRepsHabit:
			stats, err = models.GetSetRepsHabitStats(db, habitID)
		default:
			sendResponse(http.StatusBadRequest, false, "Unsupported habit type", nil)
			return
		}

		if err != nil {
			log.Printf("Error getting habit stats: %v", err)
			sendResponse(http.StatusInternalServerError, false, err.Error(), nil)
			return
		}

		// Send success response
		sendResponse(http.StatusOK, true, "", stats)
	}
}
