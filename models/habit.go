package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type HabitType string

const (
	BinaryHabit       HabitType = "binary"
	NumericHabit      HabitType = "numeric"
	OptionSelectHabit HabitType = "option-select"
	SetRepsHabit      HabitType = "set-reps"
)

type Habit struct {
	ID           int            `json:"id"`
	UserID       int            `json:"user_id"`
	Name         string         `json:"name"`
	Emoji        string         `json:"emoji"`
	CreatedAt    time.Time      `json:"created_at"`
	HabitType    HabitType      `json:"habit_type"`
	IsDefault    bool           `json:"is_default"`
	DisplayOrder int            `json:"display_order"`
	HabitOptions sql.NullString `json:"habit_options"`
}

type HabitOption struct {
	Emoji string `json:"emoji"`
	Label string `json:"label"`
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
            emoji TEXT NOT NULL DEFAULT '✨',
            habit_type TEXT NOT NULL CHECK(habit_type IN ('binary', 'numeric', 'option-select', 'set-reps')),
            is_default BOOLEAN NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            display_order INTEGER NOT NULL DEFAULT 0,
            habit_options TEXT,
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

	case OptionSelectHabit:
		// Add handling for option-select type
		_, err = db.Exec("DELETE FROM habit_logs WHERE habit_id = ? AND date = ?", hl.HabitID, hl.Date)
		if err != nil {
			return err
		}

		// Insert new log with the option value
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

	case SetRepsHabit:
		// Add handling for set-reps type
		log.Printf("SetRepsHabit: Processing log with status: %s and value: %s", hl.Status, hl.Value.String)

		_, err = db.Exec("DELETE FROM habit_logs WHERE habit_id = ? AND date = ?", hl.HabitID, hl.Date)
		if err != nil {
			log.Printf("SetRepsHabit: Error deleting existing log: %v", err)
			return err
		}

		// Skip validation for missed/skipped status
		if hl.Status == "missed" || hl.Status == "skipped" {
			log.Printf("SetRepsHabit: Handling %s status", hl.Status)

			// Insert new log with empty sets
			result, err := db.Exec(`
				INSERT INTO habit_logs (habit_id, date, status, value, created_at) 
				VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			`, hl.HabitID, hl.Date, hl.Status, hl.Value)

			if err != nil {
				log.Printf("SetRepsHabit: Error inserting %s log: %v", hl.Status, err)
				return err
			}

			id, err := result.LastInsertId()
			if err != nil {
				log.Printf("SetRepsHabit: Error getting last insert ID: %v", err)
				return err
			}

			hl.ID = int(id)
			log.Printf("SetRepsHabit: Successfully saved %s log with ID %d", hl.Status, hl.ID)
			return nil
		}

		// For 'done' status, validate sets
		var setRepsValue SetRepsValue
		if err := json.Unmarshal([]byte(hl.Value.String), &setRepsValue); err != nil {
			return err
		}

		if len(setRepsValue.Sets) == 0 {
			return fmt.Errorf("At least one set is required")
		}

		// Insert new log with the set-reps value
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
	// Insert the new habit

	result, err := db.Exec(`
    INSERT INTO habits (user_id, name, emoji, habit_type, is_default, created_at, habit_options) 
    VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
	`, h.UserID, h.Name, h.Emoji, h.HabitType, h.IsDefault, h.HabitOptions)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	h.ID = int(id)

	// Get the current max display_order for this user
	var maxOrder int
	err = db.QueryRow("SELECT IFNULL(MAX(display_order), 0) FROM habits WHERE user_id = ?", h.UserID).Scan(&maxOrder)
	if err != nil {
		return err
	}

	// Set this habit's display_order to maxOrder + 1
	h.DisplayOrder = maxOrder + 1
	_, err = db.Exec("UPDATE habits SET display_order = ? WHERE id = ?", h.DisplayOrder, h.ID)
	if err != nil {
		return err
	}

	return nil
}

// GetHabitByID retrieves a habit from the database by its ID
func GetHabitByID(db *sql.DB, id int) (*Habit, error) {
	habit := &Habit{}
	err := db.QueryRow(`
		SELECT id, user_id, name, emoji, habit_type, is_default, created_at 
		FROM habits 
		WHERE id = ?
	`, id).Scan(&habit.ID, &habit.UserID, &habit.Name, &habit.Emoji, &habit.HabitType, &habit.IsDefault, &habit.CreatedAt)

	if err != nil {
		return nil, err
	}
	return habit, nil
}

// Update modifies an existing habit in the database
func (h *Habit) Update(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE habits 
		SET name = ?, emoji = ?, habit_type = ?, is_default = ? 
		WHERE id = ?
	`, h.Name, h.Emoji, h.HabitType, h.IsDefault, h.ID)

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
		SELECT id, user_id, name, emoji, habit_type, is_default, created_at, display_order, habit_options
		FROM habits 
		WHERE user_id = ?
		ORDER BY display_order ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var habit Habit
		err := rows.Scan(
			&habit.ID,
			&habit.UserID,
			&habit.Name,
			&habit.Emoji,
			&habit.HabitType,
			&habit.IsDefault,
			&habit.CreatedAt,
			&habit.DisplayOrder,
			&habit.HabitOptions,
		)
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
	case "option-select":
		// Add validation for option-select
		var valueMap struct {
			Emoji string `json:"emoji"`
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(hl.Value.String), &valueMap); err != nil {
			return fmt.Errorf("invalid option-select value format: %v", err)
		}
		if valueMap.Emoji == "" || valueMap.Label == "" {
			return fmt.Errorf("option-select value must have both emoji and label")
		}
	case "set-reps":
		var setRepsValue SetRepsValue
		if err := json.Unmarshal([]byte(hl.Value.String), &setRepsValue); err != nil {
			return fmt.Errorf("invalid set-reps value format: %v", err)
		}

		// Allow empty sets array for missed/skipped status
		if hl.Status == "missed" || hl.Status == "skipped" {
			if len(setRepsValue.Sets) > 0 {
				return fmt.Errorf("missed/skipped status should have empty sets")
			}
			return nil
		}

		// For 'done' status, require at least one set
		if len(setRepsValue.Sets) == 0 {
			return fmt.Errorf("at least one set is required for done status")
		}

		// Validate each set
		for _, set := range setRepsValue.Sets {
			if set.Reps <= 0 {
				return fmt.Errorf("reps must be greater than 0")
			}
		}
	}

	return nil
}

func MarshalHabitOptions(options []HabitOption) (sql.NullString, error) {
	if len(options) == 0 {
		return sql.NullString{Valid: false}, nil
	}
	data, err := json.Marshal(options)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(data), Valid: true}, nil
}

type SetRep struct {
	Set   int     `json:"set"`
	Reps  int     `json:"reps"`
	Value float64 `json:"value,omitempty"` // Optional weight value
}

type SetRepsValue struct {
	Sets []SetRep `json:"sets"`
	Unit string   `json:"unit,omitempty"` // kg or lbs
}
