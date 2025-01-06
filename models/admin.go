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
		SELECT id, first_name, last_name, email, created_at 
		FROM users 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
