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

type CreateGoalRequest struct {
	HabitID      int     `json:"habit_id"`
	Name         string  `json:"name"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	TargetNumber float64 `json:"target_number"`
}

type UpdateGoalRequest struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	TargetNumber float64 `json:"target_number"`
}

type ReorderGoalsRequest struct {
	Goals []struct {
		ID       int `json:"id"`
		Position int `json:"position"`
	} `json:"goals"`
}

// GetGoalsHandler returns all goals for the authenticated user
func GetGoalsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		goals, err := models.GetGoalsByUser(db, userID)
		if err != nil {
			log.Printf("Error getting goals: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error getting goals",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    goals,
		})
	}
}

// CreateGoalHandler creates a new goal
func CreateGoalHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		userID := middleware.GetUserID(r)
		goal := &models.Goal{
			UserID:       userID,
			HabitID:      req.HabitID,
			Name:         req.Name,
			StartDate:    req.StartDate,
			EndDate:      req.EndDate,
			TargetNumber: req.TargetNumber,
		}

		if err := goal.Create(db); err != nil {
			log.Printf("Error creating goal: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error creating goal",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Goal created successfully",
			Data:    goal,
		})
	}
}

// UpdateGoalHandler updates an existing goal
func UpdateGoalHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UpdateGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		userID := middleware.GetUserID(r)
		goal, err := models.GetGoal(db, req.ID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Goal not found",
			})
			return
		}

		if goal.UserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to goal",
			})
			return
		}

		goal.Name = req.Name
		goal.StartDate = req.StartDate
		goal.EndDate = req.EndDate
		goal.TargetNumber = req.TargetNumber

		if err := goal.Update(db); err != nil {
			log.Printf("Error updating goal: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error updating goal",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Goal updated successfully",
			Data:    goal,
		})
	}
}

// DeleteGoalHandler deletes a goal
func DeleteGoalHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		goalID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid goal ID",
			})
			return
		}

		userID := middleware.GetUserID(r)
		goal, err := models.GetGoal(db, goalID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Goal not found",
			})
			return
		}

		if goal.UserID != userID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Unauthorized access to goal",
			})
			return
		}

		if err := goal.Delete(db); err != nil {
			log.Printf("Error deleting goal: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error deleting goal",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Goal deleted successfully",
		})
	}
}

// ReorderGoalsHandler updates the position of goals
func ReorderGoalsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReorderGoalsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		userID := middleware.GetUserID(r)
		for _, g := range req.Goals {
			goal, err := models.GetGoal(db, g.ID)
			if err != nil || goal.UserID != userID {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Message: "Unauthorized access to goal",
				})
				return
			}
		}

		if err := models.UpdateGoalPositions(db, req.Goals); err != nil {
			log.Printf("Error reordering goals: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Message: "Error reordering goals",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Message: "Goals reordered successfully",
		})
	}
}
