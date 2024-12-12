package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GetBinaryHabitStats retrieves statistics for a binary habit
func GetBinaryHabitStats(db *sql.DB, habitID int) (BinaryHabitStats, error) {
	// First verify this is a binary habit
	var habitType HabitType
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", habitID).Scan(&habitType)
	if err != nil {
		return BinaryHabitStats{}, fmt.Errorf("habit not found: %v", err)
	}
	if habitType != BinaryHabit {
		return BinaryHabitStats{}, fmt.Errorf("habit is not binary type")
	}

	var stats BinaryHabitStats
	var startDateStr sql.NullString

	err = db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN status = 'done' THEN 1 END) as total_done,
			COUNT(CASE WHEN status = 'missed' THEN 1 END) as total_missed,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) as total_skipped,
			COUNT(*) as total_days,
			strftime('%Y-%m-%d', MIN(CASE WHEN status = 'done' THEN date END)) as start_date
		FROM habit_logs 
		WHERE habit_id = ?
	`, habitID).Scan(
		&stats.TotalDone,
		&stats.TotalMissed,
		&stats.TotalSkipped,
		&stats.TotalDays,
		&startDateStr,
	)
	if err != nil {
		return BinaryHabitStats{}, fmt.Errorf("error getting habit stats: %v", err)
	}

	// Convert the date string to time.Time if it's not null
	if startDateStr.Valid {
		parsedTime, err := time.Parse("2006-01-02", startDateStr.String)
		if err != nil {
			return BinaryHabitStats{}, fmt.Errorf("error parsing start date: %v", err)
		}
		stats.StartDate = parsedTime
	}

	return stats, nil
}

// NumericHabitStats represents statistics for a numeric habit
type NumericHabitStats struct {
	TotalDone     int       `json:"total_done"`
	TotalReps     int       `json:"total_reps"`
	AveragePerDay float64   `json:"average_per_day"`
	TotalDays     int       `json:"total_days"`
	TotalMissed   int       `json:"total_missed"`
	TotalSkipped  int       `json:"total_skipped"`
	BiggestDay    int       `json:"biggest_day"`
	StartDate     time.Time `json:"start_date,omitempty"`
}

// GetNumericHabitStats retrieves statistics for a numeric habit
func GetNumericHabitStats(db *sql.DB, habitID int) (NumericHabitStats, error) {
	// Verify this is a numeric habit
	var habitType HabitType
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", habitID).Scan(&habitType)
	if err != nil {
		return NumericHabitStats{}, fmt.Errorf("habit not found: %v", err)
	}
	if habitType != NumericHabit {
		return NumericHabitStats{}, fmt.Errorf("habit is not numeric type")
	}

	var stats NumericHabitStats
	var startDateStr sql.NullString

	err = db.QueryRow(`
		SELECT 
			COUNT(DISTINCT CASE WHEN json_extract(value, '$.value') > 0 THEN date END) as total_done,
			COALESCE(SUM(CAST(json_extract(value, '$.value') AS INTEGER)), 0) as total_reps,
			COALESCE(
				ROUND(
					CAST(COALESCE(SUM(CAST(json_extract(value, '$.value') AS INTEGER)), 0) AS FLOAT) / 
					NULLIF(COUNT(DISTINCT CASE WHEN json_extract(value, '$.value') > 0 THEN date END), 0),
				2),
				0
			) as average_per_day,
			COUNT(DISTINCT date) as total_days,
			COUNT(CASE WHEN status = 'missed' THEN 1 END) as total_missed,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) as total_skipped,
			COALESCE(MAX(CAST(json_extract(value, '$.value') AS INTEGER)), 0) as biggest_day,
			strftime('%Y-%m-%d', MIN(CASE WHEN json_extract(value, '$.value') > 0 THEN date END)) as start_date
		FROM habit_logs 
		WHERE habit_id = ?
	`, habitID).Scan(
		&stats.TotalDone,
		&stats.TotalReps,
		&stats.AveragePerDay,
		&stats.TotalDays,
		&stats.TotalMissed,
		&stats.TotalSkipped,
		&stats.BiggestDay,
		&startDateStr,
	)
	if err != nil {
		return NumericHabitStats{}, fmt.Errorf("error getting habit stats: %v", err)
	}

	// Convert the date string to time.Time if it's not null
	if startDateStr.Valid {
		parsedTime, err := time.Parse("2006-01-02", startDateStr.String)
		if err != nil {
			return NumericHabitStats{}, fmt.Errorf("error parsing start date: %v", err)
		}
		stats.StartDate = parsedTime
	}

	return stats, nil
}

type ChoiceOption struct {
	Emoji string `json:"emoji"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type ChoiceHabitStats struct {
	Options   []ChoiceOption `json:"options"`
	TotalDays int            `json:"total_days"`
	StartDate time.Time      `json:"start_date,omitempty"`
}

func GetChoiceHabitStats(db *sql.DB, habitID int) (ChoiceHabitStats, error) {
	// First verify this is a choice habit
	var habitType HabitType
	var habitOptionsStr sql.NullString
	err := db.QueryRow(`
		SELECT habit_type, habit_options 
		FROM habits 
		WHERE id = ?`, habitID).Scan(&habitType, &habitOptionsStr)

	if err != nil {
		return ChoiceHabitStats{}, fmt.Errorf("habit not found: %v", err)
	}

	if habitType != OptionSelectHabit {
		return ChoiceHabitStats{}, fmt.Errorf("habit is not option-select type")
	}

	// Parse habit options
	var options []HabitOption
	if err := json.Unmarshal([]byte(habitOptionsStr.String), &options); err != nil {
		return ChoiceHabitStats{}, fmt.Errorf("invalid habit options format: %v", err)
	}

	// Initialize stats
	stats := ChoiceHabitStats{
		Options: make([]ChoiceOption, len(options)),
	}

	// Copy options and initialize counts
	for i, opt := range options {
		stats.Options[i] = ChoiceOption{
			Emoji: opt.Emoji,
			Label: opt.Label,
			Count: 0,
		}
	}

	// Get counts for each option and total days
	rows, err := db.Query(`
		SELECT 
			value,
			COUNT(*) as count
		FROM habit_logs 
		WHERE habit_id = ? AND status = 'done'
		GROUP BY value`, habitID)
	if err != nil {
		return ChoiceHabitStats{}, fmt.Errorf("error getting habit stats: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var valueStr string
		var count int
		if err := rows.Scan(&valueStr, &count); err != nil {
			return ChoiceHabitStats{}, fmt.Errorf("error scanning row: %v", err)
		}

		var value struct {
			Emoji string `json:"emoji"`
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			continue // Skip invalid JSON
		}

		// Find matching option and update count
		for i := range stats.Options {
			if stats.Options[i].Emoji == value.Emoji && stats.Options[i].Label == value.Label {
				stats.Options[i].Count = count
				break
			}
		}
	}

	// Get total days and start date
	var startDateStr sql.NullString
	err = db.QueryRow(`
		SELECT 
			COUNT(DISTINCT date) as total_days,
			MIN(date) as start_date
		FROM habit_logs 
		WHERE habit_id = ?`, habitID).Scan(&stats.TotalDays, &startDateStr)
	if err != nil {
		return ChoiceHabitStats{}, fmt.Errorf("error getting total days: %v", err)
	}

	// Parse start date if available
	if startDateStr.Valid {
		// First split the string to get just the date part
		datePart := strings.Split(startDateStr.String, " ")[0]
		parsedTime, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			return ChoiceHabitStats{}, fmt.Errorf("error parsing start date: %v", err)
		}
		stats.StartDate = parsedTime
	}

	return stats, nil
}
