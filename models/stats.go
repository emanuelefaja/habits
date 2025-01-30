package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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
			strftime('%Y-%m-%d', MIN(CASE WHEN status = 'done' THEN date END)) as start_date,
			(
				WITH ordered_dates AS (
					SELECT date,
						   LAG(date) OVER (ORDER BY date) AS prev_date
					FROM habit_logs
					WHERE habit_id = ? AND status = 'done'
				),
				streaks AS (
					SELECT SUM(CASE WHEN JULIANDAY(date) - JULIANDAY(prev_date) > 1 THEN 1 ELSE 0 END) 
							  OVER (ORDER BY date) AS streak_group
					FROM ordered_dates
				)
				SELECT MAX(streak_length)
				FROM (
					SELECT streak_group, COUNT(*) as streak_length
					FROM streaks
					GROUP BY streak_group
				)
			) as longest_streak
		FROM habit_logs 
		WHERE habit_id = ?
	`, habitID, habitID).Scan(
		&stats.TotalDone,
		&stats.TotalMissed,
		&stats.TotalSkipped,
		&stats.TotalDays,
		&startDateStr,
		&stats.LongestStreak,
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
	TotalDone      int       `json:"total_done"`
	TotalReps      int       `json:"total_reps"`
	AveragePerDay  float64   `json:"average_per_day"`
	TotalDays      int       `json:"total_days"`
	TotalMissed    int       `json:"total_missed"`
	TotalSkipped   int       `json:"total_skipped"`
	BiggestDay     int       `json:"biggest_day"`
	BiggestDayDate time.Time `json:"biggest_day_date,omitempty"`
	StartDate      time.Time `json:"start_date,omitempty"`
	LongestStreak  int       `json:"longest_streak"`
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
			(
				SELECT date
				FROM habit_logs
				WHERE habit_id = ?
				AND status = 'done'
				AND CAST(json_extract(value, '$.value') AS INTEGER) = (
					SELECT MAX(CAST(json_extract(value, '$.value') AS INTEGER))
					FROM habit_logs
					WHERE habit_id = ?
					AND status = 'done'
				)
				LIMIT 1
			) as biggest_day_date,
			strftime('%Y-%m-%d', MIN(CASE WHEN json_extract(value, '$.value') > 0 THEN date END)) as start_date
		FROM habit_logs 
		WHERE habit_id = ?
	`, habitID, habitID, habitID).Scan(
		&stats.TotalDone,
		&stats.TotalReps,
		&stats.AveragePerDay,
		&stats.TotalDays,
		&stats.TotalMissed,
		&stats.TotalSkipped,
		&stats.BiggestDay,
		&stats.BiggestDayDate,
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

	// Update the SQL query to calculate the longest streak
	err = db.QueryRow(`
		WITH ordered_dates AS (
			SELECT date,
				   LAG(date) OVER (ORDER BY date) AS prev_date
			FROM habit_logs
			WHERE habit_id = ? AND status = 'done'
		),
		streaks AS (
			SELECT SUM(CASE WHEN JULIANDAY(date) - JULIANDAY(prev_date) > 1 THEN 1 ELSE 0 END) 
					  OVER (ORDER BY date) AS streak_group
			FROM ordered_dates
		)
		SELECT MAX(streak_length)
		FROM (
			SELECT streak_group, COUNT(*) as streak_length
			FROM streaks
			GROUP BY streak_group
		)
	`, habitID).Scan(&stats.LongestStreak)
	if err != nil {
		return NumericHabitStats{}, fmt.Errorf("error getting habit stats: %v", err)
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

// SetRepsHabitStats represents statistics for a set-reps habit
type SetRepsHabitStats struct {
	TotalDays         int       `json:"total_days"`
	TotalSets         int       `json:"total_sets"`
	TotalReps         int       `json:"total_reps"`
	AverageRepsPerSet float64   `json:"average_reps_per_set"`
	HighestRepsInSet  int       `json:"highest_reps_in_set"`
	BestSetDate       time.Time `json:"best_set_date,omitempty"`
	AverageSetsPerDay float64   `json:"average_sets_per_day"`
	AverageRepsPerDay float64   `json:"average_reps_per_day"`
	BiggestDay        int       `json:"biggest_day"`
	BiggestDayDate    time.Time `json:"biggest_day_date,omitempty"`
	LongestStreak     int       `json:"longest_streak"`
	TotalMissed       int       `json:"total_missed"`
	TotalSkipped      int       `json:"total_skipped"`
	StartDate         time.Time `json:"start_date,omitempty"`
}

func GetSetRepsHabitStats(db *sql.DB, habitID int) (SetRepsHabitStats, error) {
	// First verify this is a set-reps habit
	var habitType HabitType
	err := db.QueryRow("SELECT habit_type FROM habits WHERE id = ?", habitID).Scan(&habitType)
	if err != nil {
		return SetRepsHabitStats{}, fmt.Errorf("habit not found: %v", err)
	}
	if habitType != SetRepsHabit {
		return SetRepsHabitStats{}, fmt.Errorf("habit is not set-reps type")
	}

	var stats SetRepsHabitStats
	var startDateStr sql.NullString
	var biggestDayDate sql.NullTime
	var bestSetDate sql.NullTime

	/*
	   Explanation of sub-selects:

	   1) total_days = COUNT(DISTINCT CASE WHEN status='done' THEN date END)
	      – Number of unique days on which the habit was done.

	   2) total_sets = SUM( CASE WHEN status='done' THEN json_array_length(...) ELSE 0 END )
	      – For each done entry, sum up the number of sets in the JSON.

	   3) total_reps =
	      SELECT SUM(json_extract(s.value, '$.reps'))
	      FROM habit_logs hl, json_each(json_extract(hl.value, '$.sets')) AS s
	      WHERE hl.habit_id = ?
	      AND hl.status = 'done'
	      – Sum of all reps across all sets for all done entries.

	   4) biggest_day =
	      SELECT MAX( sum_of_reps_in_that_day )
	      – The largest single-day total of reps.
	      (We do it by subselecting the SUM(...).)

	   5) highest_reps_in_set =
	      SELECT MAX( CAST(json_extract(s.value, '$.reps') as integer) )
	      – The largest single set's reps, across all days.

	   6) total_missed, total_skipped =
	      SELECT COUNT(*) with a CASE filter for each status.

	   7) start_date = MIN(date) for done status
	      – The earliest done date in YYYY-MM-DD format.

	   We combine them all in one SELECT so we only do one round-trip.
	*/

	query := `
		SELECT 
			COUNT(DISTINCT CASE WHEN status = 'done' THEN date END) AS total_days,

			SUM(
				CASE 
					WHEN status = 'done' THEN 
						json_array_length(json_extract(value, '$.sets'))
					ELSE 0
				END
			) AS total_sets,

			(
				SELECT SUM(json_extract(s.value, '$.reps'))
				FROM habit_logs hl,
					 json_each(json_extract(hl.value, '$.sets')) AS s
				WHERE hl.habit_id = ?
				AND hl.status = 'done'
			) AS total_reps,

			(
				WITH sets(date, reps) AS (
					SELECT hl.date, CAST(json_extract(json_each.value, '$.reps') AS INTEGER)
					FROM habit_logs hl, json_each(json_extract(hl.value, '$.sets'))
					WHERE hl.habit_id = ? AND hl.status = 'done'
				)
				SELECT COALESCE(MAX(daily_reps), 0)
				FROM (
					SELECT SUM(reps) as daily_reps
					FROM sets
					GROUP BY date
				)
			) AS biggest_day,

			(
				WITH sets(date, reps) AS (
					SELECT hl.date, CAST(json_extract(json_each.value, '$.reps') AS INTEGER)
					FROM habit_logs hl, json_each(json_extract(hl.value, '$.sets'))
					WHERE hl.habit_id = ? AND hl.status = 'done'
				)
				SELECT date
				FROM (
					SELECT date, SUM(reps) as daily_reps
					FROM sets
					GROUP BY date
				)
				WHERE daily_reps = (
					SELECT MAX(daily_reps)
					FROM (
						SELECT SUM(reps) as daily_reps
						FROM sets
						GROUP BY date
					)
				)
				LIMIT 1
			) AS biggest_day_date,

			(
				SELECT COALESCE(
					MAX(
						CAST(json_extract(s.value, '$.reps') AS INTEGER)
					), 0
				)
				FROM habit_logs hl,
					 json_each(json_extract(hl.value, '$.sets')) AS s
				WHERE hl.habit_id = ?
				AND hl.status = 'done'
			) AS highest_reps_in_set,

			(
				SELECT date
				FROM habit_logs hl,
					 json_each(json_extract(hl.value, '$.sets')) AS s
				WHERE hl.habit_id = ?
				AND hl.status = 'done'
				AND CAST(json_extract(s.value, '$.reps') AS INTEGER) = (
					SELECT MAX(CAST(json_extract(s2.value, '$.reps') AS INTEGER))
					FROM habit_logs hl2,
						 json_each(json_extract(hl2.value, '$.sets')) AS s2
					WHERE hl2.habit_id = ?
					AND hl2.status = 'done'
				)
				LIMIT 1
			) AS best_set_date,

			(
				SELECT COUNT(*)
				FROM habit_logs
				WHERE habit_id = ?
				AND status = 'missed'
			) AS total_missed,

			(
				SELECT COUNT(*)
				FROM habit_logs
				WHERE habit_id = ?
				AND status = 'skipped'
			) AS total_skipped,

			strftime('%Y-%m-%d', MIN(CASE WHEN status = 'done' THEN date END)) AS start_date

		FROM habit_logs
		WHERE habit_id = ?
	`

	err = db.QueryRow(
		query,
		// The parameters in the sub-selects must be repeated in the correct order:
		habitID, // for total_reps
		habitID, // for biggest_day
		habitID, // for biggest_day_date first subquery
		habitID, // for biggest_day_date second subquery
		habitID, // for highest_reps_in_set
		habitID, // for best_set_date first subquery
		habitID, // for best_set_date second subquery
		habitID, // for total_missed
		habitID, // for total_skipped
		habitID, // final WHERE
	).Scan(
		&stats.TotalDays,
		&stats.TotalSets,
		&stats.TotalReps,
		&stats.BiggestDay,
		&biggestDayDate,
		&stats.HighestRepsInSet,
		&bestSetDate,
		&stats.TotalMissed,
		&stats.TotalSkipped,
		&startDateStr,
	)
	if err != nil {
		return SetRepsHabitStats{}, fmt.Errorf("error getting habit stats: %v", err)
	}

	// Parse start_date if not null
	if startDateStr.Valid {
		parsedTime, err := time.Parse("2006-01-02", startDateStr.String)
		if err != nil {
			return SetRepsHabitStats{}, fmt.Errorf("error parsing start date: %v", err)
		}
		stats.StartDate = parsedTime
	}

	// Set the dates if they are valid
	if biggestDayDate.Valid {
		stats.BiggestDayDate = biggestDayDate.Time
	}
	if bestSetDate.Valid {
		stats.BestSetDate = bestSetDate.Time
	}

	// ==============
	// Post-processing for averages and longest streak
	// ==============

	// Compute derived stats in Go (you can do them in SQL if you prefer).
	if stats.TotalSets > 0 {
		stats.AverageRepsPerSet = math.Round(float64(stats.TotalReps)/float64(stats.TotalSets)*100) / 100
	}
	if stats.TotalDays > 0 {
		stats.AverageSetsPerDay = math.Round(float64(stats.TotalSets)/float64(stats.TotalDays)*100) / 100
		stats.AverageRepsPerDay = math.Round(float64(stats.TotalReps)/float64(stats.TotalDays)*100) / 100
	}

	// If you want to calculate the LongestStreak in code, you could do:
	//
	//   longestStreak, err := getLongestStreak(db, habitID)
	//   if err != nil { ... }
	//   stats.LongestStreak = longestStreak
	//
	// Where getLongestStreak is a separate helper that:
	// 1. SELECTs all done dates in ascending order
	// 2. Iterates over them to figure out the maximum run of consecutive days

	// Return the stats
	return stats, nil
}
