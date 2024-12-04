package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type HabitType string

const (
	BinaryHabit  HabitType = "binary"
	NumericHabit HabitType = "numeric"
)

type Habit struct {
	ID        int       `json:"id"`         // Unique identifier for the habit
	UserID    int       `json:"user_id"`    // ID of the user who owns this habit
	Name      string    `json:"name"`       // Name of the habit
	CreatedAt time.Time `json:"created_at"` // Timestamp of when the habit was created
	HabitType HabitType `json:"habit_type"` // Type of habit (e.g., "binary", "numeric", etc.)
	IsDefault bool      `json:"is_default"` // Flag to indicate if it's a default habit
}

type HabitLog struct {
	ID        int            `json:"id"`
	HabitID   int            `json:"habit_id"`
	Date      time.Time      `json:"date"`
	Status    string         `json:"status"`
	Value     sql.NullString `json:"value"` // JSON string for type-specific data
	CreatedAt time.Time      `json:"created_at"`
}

// InitializeDB creates the habits table if it doesn't exist
func InitializeHabitsDB(db *sql.DB) error {
	// Create habits table
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS habits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            habit_type TEXT NOT NULL CHECK(habit_type IN ('binary', 'numeric')),
            is_default BOOLEAN NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(user_id) REFERENCES users(id),
            UNIQUE(user_id, name)
        )
    `)
	if err != nil {
		return err
	}

	// Create habit_logs table
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS habit_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            habit_id INTEGER NOT NULL,
            date DATE NOT NULL,
            status TEXT NOT NULL CHECK(status IN ('done', 'missed', 'skipped')),
            value TEXT,  -- JSON string for type-specific data
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(habit_id) REFERENCES habits(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		return err
	}

	// Create index for looking up logs by date range
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_habit_logs_date 
		ON habit_logs(habit_id, date)
	`)

	return err
}

// CreateOrUpdate creates or updates a habit log based on habit type
func (hl *HabitLog) CreateOrUpdate(db *sql.DB) error {
	// Get the habit type
	var habitType HabitType
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", hl.HabitID).Scan(&habitType)
	if err != nil {
		return err
	}

	switch habitType {
	case BinaryHabit:
		// For binary habits, delete any existing log first
		_, err = db.Exec("DELETE FROM habit_logs WHERE habit_id = ? AND date = ?", hl.HabitID, hl.Date)
		if err != nil {
			return err
		}

		if hl.Status != "none" {
			// Insert new log if status is not "none"
			result, err := db.Exec(`
				INSERT INTO habit_logs (habit_id, date, status, value, created_at) 
				VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			`, hl.HabitID, hl.Date, hl.Status, hl.Value)

			if err != nil {
				return err
			}

			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			hl.ID = int(id)
		}

	case NumericHabit:
		// For numeric habits, replace any existing log for this date
		_, err = db.Exec("DELETE FROM habit_logs WHERE habit_id = ? AND date = ?", hl.HabitID, hl.Date)
		if err != nil {
			return err
		}

		// Insert new log with the latest value and status
		result, err := db.Exec(`
			INSERT INTO habit_logs (habit_id, date, status, value, created_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, hl.HabitID, hl.Date, hl.Status, hl.Value)

		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		hl.ID = int(id)

	default:
		return fmt.Errorf("unknown habit type: %s", habitType)
	}

	return nil
}

// GetHabitLogsByDateRange retrieves all logs for a habit within a date range
func GetHabitLogsByDateRange(db *sql.DB, habitID int, startDate, endDate time.Time) ([]HabitLog, error) {
	logs := []HabitLog{}
	rows, err := db.Query(`
		SELECT id, habit_id, date, status, value, created_at 
		FROM habit_logs 
		WHERE habit_id = ? AND date BETWEEN ? AND ?
		ORDER BY date ASC, created_at ASC
	`, habitID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var log HabitLog
		err := rows.Scan(&log.ID, &log.HabitID, &log.Date, &log.Status, &log.Value, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
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

// Delete removes a habit log from the database
func (hl *HabitLog) Delete(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM habit_logs WHERE id = ?", hl.ID)
	return err
}

// GetHabitLogByID retrieves a single habit log by its ID
func GetHabitLogByID(db *sql.DB, id int) (*HabitLog, error) {
	log := &HabitLog{}
	err := db.QueryRow(`
		SELECT id, habit_id, date, status, value, created_at 
		FROM habit_logs 
		WHERE id = ?
	`, id).Scan(&log.ID, &log.HabitID, &log.Date, &log.Status, &log.Value, &log.CreatedAt)

	if err != nil {
		return nil, err
	}
	return log, nil
}

// SetValue sets the value field with proper JSON encoding based on habit type
func (hl *HabitLog) SetValue(value interface{}) error {
	if value == nil {
		hl.Value = sql.NullString{Valid: false}
		return nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	hl.Value = sql.NullString{
		String: string(jsonBytes),
		Valid:  true,
	}
	return nil
}

// GetValue unmarshals the JSON value into the provided interface
func (hl *HabitLog) GetValue(value interface{}) error {
	if !hl.Value.Valid {
		return nil // No value set
	}

	return json.Unmarshal([]byte(hl.Value.String), value)
}

// ValidateValue checks if the value matches the expected structure for the habit type
func (hl *HabitLog) ValidateValue(db *sql.DB) error {
	// Get habit type
	var habitType string
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", hl.HabitID).Scan(&habitType)
	if err != nil {
		return err
	}

	// For binary habits, value should be null
	if habitType == "binary" {
		if hl.Value.Valid {
			return fmt.Errorf("binary habits should not have a value")
		}
		return nil
	}

	// For other types, value must be valid JSON
	if !hl.Value.Valid {
		return fmt.Errorf("non-binary habits must have a value")
	}

	// Validate JSON structure based on habit type
	var valueMap map[string]interface{}
	if err := json.Unmarshal([]byte(hl.Value.String), &valueMap); err != nil {
		return fmt.Errorf("invalid JSON value: %v", err)
	}

	switch habitType {
	case "numeric":
		if _, ok := valueMap["value"]; !ok {
			return fmt.Errorf("numeric habits must have a 'value' field")
		}
	case "time":
		if _, ok := valueMap["minutes"]; !ok {
			return fmt.Errorf("time habits must have a 'minutes' field")
		}
	case "distance_time":
		if _, ok := valueMap["distance"]; !ok {
			return fmt.Errorf("distance_time habits must have a 'distance' field")
		}
		if _, ok := valueMap["minutes"]; !ok {
			return fmt.Errorf("distance_time habits must have a 'minutes' field")
		}
		if _, ok := valueMap["unit"]; !ok {
			return fmt.Errorf("distance_time habits must have a 'unit' field")
		}
	}

	return nil
}
