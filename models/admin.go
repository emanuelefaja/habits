package models

import "database/sql"

func GetTotalUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTotalHabits(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM habits").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTotalGoals(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM goals").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTotalHabitLogs(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM habit_logs").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetAllUsers(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`
		SELECT 
			users.id, 
			users.first_name, 
			users.last_name, 
			users.email, 
			users.created_at,
			COUNT(DISTINCT habits.id) AS habits_count,
			COUNT(DISTINCT habit_logs.id) AS logs_count
		FROM users
		LEFT JOIN habits ON users.id = habits.user_id
		LEFT JOIN habit_logs ON habits.id = habit_logs.habit_id
		GROUP BY users.id
		ORDER BY users.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.CreatedAt,
			&user.HabitsCount,
			&user.LogsCount,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetSignupStatus retrieves the current signup status from the database
func GetSignupStatus(db *sql.DB) (bool, error) {
	// Check if the settings table exists
	var tableExists int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='settings'").Scan(&tableExists)
	if err != nil {
		return true, err
	}

	// Create settings table if it doesn't exist
	if tableExists == 0 {
		_, err = db.Exec(`CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`)
		if err != nil {
			return true, err
		}

		// Default to allowing signups
		_, err = db.Exec("INSERT INTO settings (key, value) VALUES ('allow_signups', 'true')")
		if err != nil {
			return true, err
		}
		return true, nil
	}

	// Get the current signup status
	var value string
	err = db.QueryRow("SELECT value FROM settings WHERE key = 'allow_signups'").Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			// If the setting doesn't exist, create it with default value (true)
			_, err = db.Exec("INSERT INTO settings (key, value) VALUES ('allow_signups', 'true')")
			if err != nil {
				return true, err
			}
			return true, nil
		}
		return true, err
	}

	return value == "true", nil
}

// SetSignupStatus updates the signup status in the database
func SetSignupStatus(db *sql.DB, allow bool) error {
	value := "false"
	if allow {
		value = "true"
	}

	// Check if the setting exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = 'allow_signups'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update existing setting
		_, err = db.Exec("UPDATE settings SET value = ? WHERE key = 'allow_signups'", value)
	} else {
		// Insert new setting
		_, err = db.Exec("INSERT INTO settings (key, value) VALUES ('allow_signups', ?)", value)
	}

	return err
}
