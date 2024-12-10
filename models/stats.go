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
