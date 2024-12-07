package models

import (
	"database/sql"
)

// InitDB initializes the database and creates tables if they don't exist
func InitDB(db *sql.DB) error {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create habits table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS habits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			emoji TEXT NOT NULL DEFAULT '✨',
			habit_type TEXT NOT NULL CHECK(habit_type IN ('binary', 'numeric', 'option-select')),
			is_default BOOLEAN NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			display_order INTEGER NOT NULL DEFAULT 0,
			habit_options TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id),
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
			user_id INTEGER NOT NULL,
			value TEXT NOT NULL,
			date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (habit_id) REFERENCES habits(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create index on habit_logs.date
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_habit_logs_date 
		ON habit_logs(date)
	`)
	if err != nil {
		return err
	}

	// Create index on habits.user_id
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_habits_user_id 
		ON habits(user_id)
	`)
	if err != nil {
		return err
	}

	// Create sessions table (if using database sessions)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create roadmap_ideas table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roadmap_ideas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			idea_text TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create roadmap_likes table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roadmap_likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			card_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, card_id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create index for roadmap_likes
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_roadmap_likes_card_id 
		ON roadmap_likes(card_id)
	`)
	if err != nil {
		return err
	}

	return nil
}
