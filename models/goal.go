package models

import (
	"database/sql"
	"fmt"
	"time"
)

type Goal struct {
	ID            int     `json:"id"`
	UserID        int     `json:"user_id"`
	HabitID       int     `json:"habit_id"`
	Name          string  `json:"name"`
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	TargetNumber  float64 `json:"target_number"`
	CurrentNumber float64 `json:"current_number"`
	Status        string  `json:"status"`
	Position      int     `json:"position"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	HabitName     string  `json:"habit_name"`
	HabitEmoji    string  `json:"habit_emoji"`
}

// CRUD Methods

func (g *Goal) Create(db *sql.DB) error {
	query := `
		INSERT INTO goals (
			user_id, habit_id, name, start_date, end_date, 
			target_number, position
		) VALUES (?, ?, ?, ?, ?, ?, (
			SELECT COALESCE(MAX(position), 0) + 1 
			FROM goals 
			WHERE user_id = ?
		))
		RETURNING id, created_at, updated_at`

	return db.QueryRow(
		query,
		g.UserID, g.HabitID, g.Name, g.StartDate, g.EndDate,
		g.TargetNumber, g.UserID,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func GetGoal(db *sql.DB, id int) (*Goal, error) {
	goal := &Goal{}
	query := `SELECT * FROM goals WHERE id = ?`
	err := db.QueryRow(query, id).Scan(
		&goal.ID, &goal.UserID, &goal.HabitID, &goal.Name,
		&goal.StartDate, &goal.EndDate, &goal.TargetNumber,
		&goal.CurrentNumber, &goal.Status, &goal.Position,
		&goal.CreatedAt, &goal.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return goal, nil
}

func GetGoalsByUser(db *sql.DB, userID int) ([]Goal, error) {
	query := `
		SELECT 
			g.id,
			g.user_id,
			g.habit_id,
			g.name,
			g.start_date,
			g.end_date,
			g.target_number,
			g.position,
			g.created_at,
			g.updated_at,
			h.emoji as habit_emoji,
			h.name as habit_name
		FROM goals g
		JOIN habits h ON g.habit_id = h.id
		WHERE g.user_id = ?
		ORDER BY g.position, g.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		err := rows.Scan(
			&g.ID,
			&g.UserID,
			&g.HabitID,
			&g.Name,
			&g.StartDate,
			&g.EndDate,
			&g.TargetNumber,
			&g.Position,
			&g.CreatedAt,
			&g.UpdatedAt,
			&g.HabitEmoji,
			&g.HabitName,
		)
		if err != nil {
			return nil, err
		}

		// Calculate progress in memory
		if err := g.CalculateProgressInMemory(db); err != nil {
			return nil, fmt.Errorf("error calculating progress for goal %d: %v", g.ID, err)
		}

		goals = append(goals, g)
	}

	return goals, nil
}

func (g *Goal) Update(db *sql.DB) error {
	query := `
		UPDATE goals 
		SET name = ?, start_date = ?, end_date = ?, 
			target_number = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?`

	result, err := db.Exec(query,
		g.Name, g.StartDate, g.EndDate,
		g.TargetNumber, g.ID, g.UserID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (g *Goal) Delete(db *sql.DB) error {
	query := `DELETE FROM goals WHERE id = ? AND user_id = ?`
	result, err := db.Exec(query, g.ID, g.UserID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdatePosition updates the position of a goal and reorders other goals
func (g *Goal) UpdatePosition(db *sql.DB, newPosition int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update positions of other goals
	if newPosition > g.Position {
		_, err = tx.Exec(`
			UPDATE goals 
			SET position = position - 1 
			WHERE user_id = ? AND position > ? AND position <= ?`,
			g.UserID, g.Position, newPosition,
		)
	} else {
		_, err = tx.Exec(`
			UPDATE goals 
			SET position = position + 1 
			WHERE user_id = ? AND position >= ? AND position < ?`,
			g.UserID, newPosition, g.Position,
		)
	}
	if err != nil {
		return err
	}

	// Update position of current goal
	_, err = tx.Exec(`
		UPDATE goals 
		SET position = ? 
		WHERE id = ?`,
		newPosition, g.ID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateStatus updates the status of a goal
func (g *Goal) UpdateStatus(db *sql.DB) error {
	query := `
		UPDATE goals 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`

	result, err := db.Exec(query, g.Status, g.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ValidateHabitType checks if the habit type is valid for goal creation
func (g *Goal) ValidateHabitType(db *sql.DB) error {
	var habitType string
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ? AND user_id = ?",
		g.HabitID, g.UserID).Scan(&habitType)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("habit not found or unauthorized")
		}
		return fmt.Errorf("error checking habit type: %v", err)
	}

	if habitType == "option-select" {
		return fmt.Errorf("goals cannot be created for option-select habits")
	}
	return nil
}

// Validate checks if the goal data is valid
func (g *Goal) Validate() error {
	// Parse dates
	startDate, err := time.Parse("2006-01-02", g.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date format: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", g.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date format: %v", err)
	}

	// Check date order
	if !endDate.After(startDate) {
		return fmt.Errorf("end date must be after start date")
	}

	// Check target number
	if g.TargetNumber <= 0 {
		return fmt.Errorf("target number must be positive")
	}

	return nil
}

// CalculateProgress updates the current progress and status of the goal
func (g *Goal) CalculateProgress(db *sql.DB) error {
	// Get habit type
	var habitType string
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", g.HabitID).Scan(&habitType)
	if err != nil {
		return fmt.Errorf("error getting habit type: %v", err)
	}

	fmt.Printf("Calculating progress for goal %d (habit %d, type %s)\n", g.ID, g.HabitID, habitType)

	// Calculate current progress based on habit type
	var query string
	switch habitType {
	case "binary":
		query = `
			SELECT COUNT(DISTINCT date) 
			FROM habit_logs 
			WHERE habit_id = ? 
			AND date BETWEEN ? AND ? 
			AND status = 'done'`
	case "numeric":
		query = `
			SELECT COALESCE(SUM(CAST(json_extract(value, '$.value') AS FLOAT)), 0)
			FROM habit_logs 
			WHERE habit_id = ? 
			AND date BETWEEN ? AND ? 
			AND status = 'done'`
	case "set-reps":
		query = `
			SELECT COALESCE(
				(
					SELECT SUM(json_extract(s.value, '$.reps'))
					FROM habit_logs hl,
						 json_each(json_extract(hl.value, '$.sets')) AS s
					WHERE hl.habit_id = ?
					AND date(hl.date) BETWEEN date(?) AND date(?)
					AND hl.status = 'done'
				),
				0
			)`
	default:
		return fmt.Errorf("unsupported habit type: %s", habitType)
	}

	err = db.QueryRow(query, g.HabitID, g.StartDate, g.EndDate).Scan(&g.CurrentNumber)
	if err != nil {
		return fmt.Errorf("error calculating progress: %v", err)
	}

	// Calculate expected progress
	startDate, err := time.Parse("2006-01-02", g.StartDate)
	if err != nil {
		return fmt.Errorf("error parsing start date: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", g.EndDate)
	if err != nil {
		return fmt.Errorf("error parsing end date: %v", err)
	}

	// Get today's date in UTC to ensure consistent comparison
	todayTime := time.Now().UTC().Truncate(24 * time.Hour)
	endDate = endDate.UTC().Truncate(24 * time.Hour)

	// Store whether today is past the end date
	isPastEndDate := todayTime.After(endDate)

	// For progress calculation, cap today at end date
	if isPastEndDate {
		todayTime = endDate
	}

	totalDays := endDate.Sub(startDate).Hours() / 24
	daysPassed := todayTime.Sub(startDate).Hours() / 24
	expectedProgress := (daysPassed / totalDays) * g.TargetNumber

	// Determine status
	switch {
	case g.CurrentNumber >= g.TargetNumber:
		g.Status = "done"
	case isPastEndDate:
		g.Status = "failed"
	case g.CurrentNumber >= expectedProgress:
		g.Status = "on_track"
	case g.CurrentNumber >= expectedProgress*0.9:
		g.Status = "at_risk"
	default:
		g.Status = "off_track"
	}

	fmt.Printf("Current progress: %f/%f\n", g.CurrentNumber, g.TargetNumber)

	// After calculating current_number, update it in the database along with the status
	_, err = db.Exec(`
		UPDATE goals 
		SET current_number = ?, status = ?
		WHERE id = ?`,
		g.CurrentNumber, g.Status, g.ID)
	if err != nil {
		return fmt.Errorf("error updating goal progress: %v", err)
	}

	return nil
}

// GetGoalsByHabit returns all active goals for a given habit
func GetGoalsByHabit(db *sql.DB, habitID int) ([]*Goal, error) {
	goals := []*Goal{}
	rows, err := db.Query(`
		SELECT id, user_id, habit_id, name, start_date, end_date, 
			   target_number, position,
			   created_at, updated_at
		FROM goals 
		WHERE habit_id = ? 
		AND end_date >= DATE('now')
		ORDER BY position ASC`, habitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		goal := &Goal{}
		err := rows.Scan(
			&goal.ID, &goal.UserID, &goal.HabitID, &goal.Name,
			&goal.StartDate, &goal.EndDate, &goal.TargetNumber,
			&goal.Position, &goal.CreatedAt, &goal.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Calculate progress in memory
		if err := goal.CalculateProgressInMemory(db); err != nil {
			return nil, fmt.Errorf("error calculating progress for goal %d: %v", goal.ID, err)
		}

		goals = append(goals, goal)
	}
	return goals, nil
}

// UpdateGoalPositions updates the positions of multiple goals in a single transaction
func UpdateGoalPositions(db *sql.DB, goals []struct {
	ID       int `json:"id"`
	Position int `json:"position"`
}) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, g := range goals {
		_, err := tx.Exec(`
			UPDATE goals 
			SET position = ?, 
				updated_at = CURRENT_TIMESTAMP 
			WHERE id = ?`,
			g.Position, g.ID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func GetGoalsByUserWithHabitInfo(db *sql.DB, userID int) ([]Goal, error) {
	query := `
		SELECT 
			g.id,
			g.user_id,
			g.habit_id,
			g.name,
			g.start_date,
			g.end_date,
			g.target_number,
			g.position,
			g.created_at,
			g.updated_at,
			h.name as habit_name,
			h.emoji as habit_emoji
		FROM goals g
		JOIN habits h ON g.habit_id = h.id
		WHERE g.user_id = ?
		ORDER BY g.position`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		err := rows.Scan(
			&g.ID,
			&g.UserID,
			&g.HabitID,
			&g.Name,
			&g.StartDate,
			&g.EndDate,
			&g.TargetNumber,
			&g.Position,
			&g.CreatedAt,
			&g.UpdatedAt,
			&g.HabitName,
			&g.HabitEmoji,
		)
		if err != nil {
			return nil, err
		}

		// Calculate progress in memory
		if err := g.CalculateProgressInMemory(db); err != nil {
			return nil, fmt.Errorf("error calculating progress for goal %d: %v", g.ID, err)
		}

		goals = append(goals, g)
	}
	return goals, nil
}

// CalculateProgressInMemory calculates the current progress and status of the goal without writing to the database
func (g *Goal) CalculateProgressInMemory(db *sql.DB) error {
	// Get habit type
	var habitType string
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", g.HabitID).Scan(&habitType)
	if err != nil {
		return fmt.Errorf("error getting habit type: %v", err)
	}

	// Calculate current progress based on habit type
	var query string
	switch habitType {
	case "binary":
		query = `
			SELECT COUNT(DISTINCT date) 
			FROM habit_logs 
			WHERE habit_id = ? 
			AND date BETWEEN ? AND ? 
			AND status = 'done'`
	case "numeric":
		query = `
			SELECT COALESCE(SUM(CAST(json_extract(value, '$.value') AS FLOAT)), 0)
			FROM habit_logs 
			WHERE habit_id = ? 
			AND date BETWEEN ? AND ? 
			AND status = 'done'`
	case "set-reps":
		query = `
			SELECT COALESCE(
				(
					SELECT SUM(json_extract(s.value, '$.reps'))
					FROM habit_logs hl,
						 json_each(json_extract(hl.value, '$.sets')) AS s
					WHERE hl.habit_id = ?
					AND date(hl.date) BETWEEN date(?) AND date(?)
					AND hl.status = 'done'
				),
				0
			)`
	default:
		return fmt.Errorf("unsupported habit type: %s", habitType)
	}

	err = db.QueryRow(query, g.HabitID, g.StartDate, g.EndDate).Scan(&g.CurrentNumber)
	if err != nil {
		return fmt.Errorf("error calculating progress: %v", err)
	}

	// Calculate expected progress
	startDate, err := time.Parse("2006-01-02", g.StartDate)
	if err != nil {
		return fmt.Errorf("error parsing start date: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", g.EndDate)
	if err != nil {
		return fmt.Errorf("error parsing end date: %v", err)
	}

	// Get today's date in UTC to ensure consistent comparison
	todayTime := time.Now().UTC().Truncate(24 * time.Hour)
	endDate = endDate.UTC().Truncate(24 * time.Hour)

	// Store whether today is past the end date
	isPastEndDate := todayTime.After(endDate)

	// For progress calculation, cap today at end date
	if isPastEndDate {
		todayTime = endDate
	}

	totalDays := endDate.Sub(startDate).Hours() / 24
	daysPassed := todayTime.Sub(startDate).Hours() / 24
	expectedProgress := (daysPassed / totalDays) * g.TargetNumber

	// Determine status
	switch {
	case g.CurrentNumber >= g.TargetNumber:
		g.Status = "done"
	case isPastEndDate:
		g.Status = "failed"
	case g.CurrentNumber >= expectedProgress:
		g.Status = "on_track"
	case g.CurrentNumber >= expectedProgress*0.9:
		g.Status = "at_risk"
	default:
		g.Status = "off_track"
	}

	return nil
}
