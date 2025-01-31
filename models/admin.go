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
