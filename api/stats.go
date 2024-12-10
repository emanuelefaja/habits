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

		// Get habit ID from URL query parameter
		idStr := r.URL.Query().Get("id")
		habitID, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid habit ID",
			})
			return
		}

		// Add debug logging
		log.Printf("Getting stats for habit ID: %d", habitID)

		// Verify habit belongs to user
		userID := middleware.GetUserID(r)
		var habitUserID int
		err = db.QueryRow("SELECT user_id FROM habits WHERE id = ?", habitID).Scan(&habitUserID)
		if err != nil {
			log.Printf("Error getting habit user ID: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error getting habit",
			})
			return
		}

		if habitUserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to habit",
			})
			return
		}

		// Get habit type
		var habitType models.HabitType
		err = db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", habitID).Scan(&habitType)
		if err != nil {
			log.Printf("Error getting habit type: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error getting habit type",
			})
			return
		}

		var stats interface{}
		switch habitType {
		case models.BinaryHabit:
			stats, err = models.GetBinaryHabitStats(db, habitID)
		case models.NumericHabit:
			stats, err = models.GetNumericHabitStats(db, habitID)
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unsupported habit type",
			})
			return
		}

		if err != nil {
			log.Printf("Error getting habit stats: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Log the stats we're about to send
		log.Printf("Returning stats for habit %d: %+v", habitID, stats)

		// Return success response
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    stats,
		}); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}
}
