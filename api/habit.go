package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"mad/models"
)

// CreateHabitHandler handles the creation of a new habit
func CreateHabitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var habit models.Habit
		if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := habit.Create(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(habit)
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
