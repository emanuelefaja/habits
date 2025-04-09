package models

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
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
			show_confetti BOOLEAN NOT NULL DEFAULT 1,
			show_weekdays BOOLEAN NOT NULL DEFAULT false,
			notification_enabled BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_admin BOOLEAN NOT NULL DEFAULT 0
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
			habit_type TEXT NOT NULL CHECK(habit_type IN ('binary', 'numeric', 'option-select', "set-reps")),
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
			date DATE NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('done', 'missed', 'skipped')),
			value TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(habit_id) REFERENCES habits(id) ON DELETE CASCADE,
			UNIQUE(habit_id, date)
		)
	`)
	if err != nil {
		return err
	}

	// Create index on habit_logs.date
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_habit_logs_date 
		ON habit_logs(habit_id, date)
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

	// Create index on habits.user_id and display_order
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_habits_user_id_display_order 
		ON habits(user_id, display_order)
	`)
	if err != nil {
		return err
	}

	// Create commits table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS commits (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			date TIMESTAMP NOT NULL,
			additions INTEGER NOT NULL,
			deletions INTEGER NOT NULL,
			files_added INTEGER NOT NULL,
			files_removed INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create index on commits.date
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_commits_date 
		ON commits(date DESC)
	`)
	if err != nil {
		return err
	}

	// Create password_reset_tokens table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			email TEXT NOT NULL,
			expiry TIMESTAMP NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Add indexes for password_reset_tokens
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_email ON password_reset_tokens(email);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id)
	`)
	if err != nil {
		return err
	}

	// Create goals table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			habit_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			target_number REAL NOT NULL,
			current_number REAL DEFAULT 0,
			status TEXT CHECK(status IN ('on_track', 'at_risk', 'off_track', 'done', 'failed')) DEFAULT 'on_track',
			position INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes for goals table
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_goals_user_id ON goals(user_id);
		CREATE INDEX IF NOT EXISTS idx_goals_habit_id ON goals(habit_id);
		CREATE INDEX IF NOT EXISTS idx_goals_position ON goals(position)
	`)
	if err != nil {
		return err
	}

	// Create settings table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Insert default settings if they don't exist
	_, err = db.Exec(`
		INSERT OR IGNORE INTO settings (key, value) 
		VALUES ('allow_signups', 'true')
	`)
	if err != nil {
		return err
	}

	// Create email_subscriptions table
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS email_subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NULL REFERENCES users(id) ON DELETE CASCADE,
		email TEXT NOT NULL,
		campaign_id TEXT NOT NULL,
		token TEXT NOT NULL,
		subscribed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT NOT NULL CHECK (status IN ('active', 'unsubscribed')) DEFAULT 'active',
		last_email_sent INTEGER DEFAULT 0,
		unsubscribed_at TIMESTAMP NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(email, campaign_id)
	)`)
	if err != nil {
		return fmt.Errorf("error creating email_subscriptions table: %w", err)
	}

	// Create email_sends table
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS email_sends (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subscription_id INTEGER NOT NULL REFERENCES email_subscriptions(id) ON DELETE CASCADE,
		email_number INTEGER NOT NULL,
		template_name TEXT NOT NULL,
		subject TEXT NOT NULL,
		status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'retry')),
		sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		error_message TEXT,
		retry_count INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("error creating email_sends table: %w", err)
	}

	// Create indexes for email campaign tables
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS idx_email_subscriptions_user_id ON email_subscriptions(user_id);
	CREATE INDEX IF NOT EXISTS idx_email_subscriptions_email ON email_subscriptions(email);
	CREATE INDEX IF NOT EXISTS idx_email_subscriptions_campaign_id ON email_subscriptions(campaign_id);
	CREATE INDEX IF NOT EXISTS idx_email_subscriptions_status ON email_subscriptions(status);
	CREATE INDEX IF NOT EXISTS idx_email_sends_subscription_id ON email_sends(subscription_id);
	CREATE INDEX IF NOT EXISTS idx_email_sends_status ON email_sends(status);
	CREATE INDEX IF NOT EXISTS idx_email_sends_sent_at ON email_sends(sent_at);
	`)
	if err != nil {
		return fmt.Errorf("error creating email campaign indexes: %w", err)
	}

	// Create user_lesson_completion table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_lesson_completion (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			lesson_id TEXT NOT NULL,
			module_id TEXT NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			completed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, lesson_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating user_lesson_completion table: %w", err)
	}

	// Create indexes for user_lesson_completion
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_lesson_completion_user_id ON user_lesson_completion(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_lesson_completion_lesson_id ON user_lesson_completion(lesson_id)
	`)
	if err != nil {
		return fmt.Errorf("error creating user_lesson_completion indexes: %w", err)
	}

	// Create user_course_access table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_course_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			course_id TEXT NOT NULL,
			purchased_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			purchase_price REAL,
			status TEXT NOT NULL CHECK(status IN ('active', 'refunded', 'expired')) DEFAULT 'active',
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, course_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating user_course_access table: %w", err)
	}

	// Create indexes for user_course_access
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_course_access_user_id ON user_course_access(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_course_access_course_id ON user_course_access(course_id)
	`)
	if err != nil {
		return fmt.Errorf("error creating user_course_access indexes: %w", err)
	}

	return nil
}

// Add this new function after the InitDB function
func SeedUsers(db *sql.DB) error {
	// Admin user
	adminPass := "adminpassword"
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (first_name, last_name, email, password_hash, is_admin, show_confetti, notification_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Admin",
		"User",
		"admin@example.com",
		string(adminHash),
		1,
		1,
		1,
	)
	if err != nil {
		return err
	}

	// Normal user
	userPass := "password"
	userHash, err := bcrypt.GenerateFromPassword([]byte(userPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (first_name, last_name, email, password_hash, is_admin, show_confetti, notification_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Normal",
		"User",
		"user@example.com",
		string(userHash),
		0,
		1,
		1,
	)

	return err
}

// MigrateDB handles database schema migrations
func MigrateDB(db *sql.DB) error {
	// Check if notification_enabled column exists in users table
	var columnExists bool
	err := db.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('users') 
		WHERE name = 'notification_enabled'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	// Add notification_enabled column if it doesn't exist
	if !columnExists {
		_, err = db.Exec(`
			ALTER TABLE users 
			ADD COLUMN notification_enabled BOOLEAN NOT NULL DEFAULT true
		`)
		if err != nil {
			return err
		}
	}

	return nil
}
