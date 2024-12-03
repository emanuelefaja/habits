package models

import (
	"database/sql"
	"time"
)

type Habit struct {
	ID        int       `json:"id"`         // Unique identifier for the habit
	UserID    int       `json:"user_id"`    // ID of the user who owns this habit
	Name      string    `json:"name"`       // Name of the habit
	CreatedAt time.Time `json:"created_at"` // Timestamp of when the habit was created
	HabitType string    `json:"habit_type"` // Type of habit (e.g., "binary", "numeric", etc.)
	IsDefault bool      `json:"is_default"` // Flag to indicate if it's a default habit
}

// InitializeDB creates the habits table if it doesn't exist
func InitializeHabitsDB(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS habits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            habit_type TEXT NOT NULL,
            is_default BOOLEAN NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(user_id) REFERENCES users(id),
            UNIQUE(user_id, name)
        )
    `)
	return err
}

// Create inserts a new habit into the database
func (h *Habit) Create(db *sql.DB) error {
	result, err := db.Exec(`
		INSERT INTO habits (user_id, name, habit_type, is_default, created_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, h.UserID, h.Name, h.HabitType, h.IsDefault)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	h.ID = int(id)
	return nil
}

// GetHabitByID retrieves a habit from the database by its ID
func GetHabitByID(db *sql.DB, id int) (*Habit, error) {
	habit := &Habit{}
	err := db.QueryRow(`
		SELECT id, user_id, name, habit_type, is_default, created_at 
		FROM habits 
		WHERE id = ?
	`, id).Scan(&habit.ID, &habit.UserID, &habit.Name, &habit.HabitType, &habit.IsDefault, &habit.CreatedAt)

	if err != nil {
		return nil, err
	}
	return habit, nil
}

// Update modifies an existing habit in the database
func (h *Habit) Update(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE habits 
		SET name = ?, habit_type = ?, is_default = ? 
		WHERE id = ?
	`, h.Name, h.HabitType, h.IsDefault, h.ID)

	return err
}

// Delete removes a habit from the database
func (h *Habit) Delete(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM habits WHERE id = ?", h.ID)
	return err
}

// HabitExists checks if a habit with the given name already exists for the user
func HabitExists(db *sql.DB, name string, userID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM habits 
			WHERE user_id = ? AND LOWER(name) = LOWER(?)
		)
	`, userID, name).Scan(&exists)
	return exists, err
}

// GetHabitsByUserID retrieves all habits for a given user
func GetHabitsByUserID(db *sql.DB, userID int) ([]Habit, error) {
	habits := []Habit{}
	rows, err := db.Query(`
		SELECT id, user_id, name, habit_type, is_default, created_at 
		FROM habits 
		WHERE user_id = ?
		ORDER BY LOWER(name) ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var habit Habit
		err := rows.Scan(&habit.ID, &habit.UserID, &habit.Name, &habit.HabitType, &habit.IsDefault, &habit.CreatedAt)
		if err != nil {
			return nil, err
		}
		habits = append(habits, habit)
	}
	return habits, nil
}
