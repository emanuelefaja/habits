package models

import (
	"database/sql"
	"fmt"
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
