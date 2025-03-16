package models

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                  int64     `json:"id"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	Email               string    `json:"email"`
	ShowConfetti        bool      `json:"show_confetti"`
	ShowWeekdays        bool      `json:"show_weekdays"`
	CreatedAt           time.Time `json:"created_at"`
	IsAdmin             bool      `json:"is_admin"`
	HabitsCount         int       `json:"habits_count"`
	LogsCount           int       `json:"logs_count"`
	NotificationEnabled bool      `json:"notification_enabled"`
}

// GetUserByID retrieves a user from the database by their ID
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, first_name, last_name, email, show_confetti, show_weekdays, created_at, is_admin, notification_enabled 
		FROM users 
		WHERE id = ?
	`, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.ShowConfetti,
		&user.ShowWeekdays,
		&user.CreatedAt,
		&user.IsAdmin,
		&user.NotificationEnabled,
	)

	if err != nil {
		return nil, err
	}

	log.Printf("GetUserByID(%d): notification_enabled=%v", id, user.NotificationEnabled)
	return user, nil
}

// GetUserByEmail retrieves a user from the database by their email
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, first_name, last_name, email, show_confetti, created_at, is_admin, notification_enabled 
		FROM users 
		WHERE email = ?
	`, email).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.ShowConfetti, &user.CreatedAt, &user.IsAdmin, &user.NotificationEnabled)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// Create inserts a new user into the database
func (u *User) Create(db *sql.DB, passwordHash string) error {
	log.Println("Attempting to create user:", u.Email)

	// Convert email to lowercase before saving
	u.Email = strings.ToLower(u.Email)

	result, err := db.Exec(`
		INSERT INTO users (first_name, last_name, email, password_hash, show_confetti, created_at, is_admin, notification_enabled) 
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?)
	`, u.FirstName, u.LastName, u.Email, passwordHash, true, false, true)

	if err != nil {
		log.Println("Error executing insert:", err)
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println("Error getting last insert ID:", err)
		return err
	}

	u.ID = id
	log.Println("User created with ID:", u.ID)
	return nil
}

// Update modifies an existing user in the database
func (u *User) Update(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE users 
		SET first_name = ?, last_name = ?, email = ?, show_confetti = ?, notification_enabled = ?
		WHERE id = ?
	`, u.FirstName, u.LastName, u.Email, u.ShowConfetti, u.NotificationEnabled, u.ID)

	return err
}

// Delete removes a user from the database
func (u *User) Delete(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", u.ID)
	return err
}

// ValidatePassword checks if the provided password matches the stored hash
func ValidatePassword(db *sql.DB, email, password string) (bool, error) {
	// Convert email to lowercase before checking
	email = strings.ToLower(email)

	var storedHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE email = ?", email).Scan(&storedHash)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil, nil
}

// InitializeDB creates the users table if it doesn't exist
func InitializeUsersDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN NOT NULL DEFAULT 0,
			show_confetti BOOLEAN NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// UpdatePassword updates the user's password hash in the database
func UpdatePassword(db *sql.DB, userID int64, currentPassword, newPassword string) error {
	// First verify the current password
	var storedHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&storedHash)
	if err != nil {
		return err
	}

	// Check if current password matches
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(currentPassword)); err != nil {
		return err
	}

	// Generate new password hash
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update the password
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, userID)
	return err
}

// DeleteUserAndData deletes the user and all associated data
func DeleteUserAndData(db *sql.DB, userID int64) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete habit logs first (due to foreign key constraints)
	_, err = tx.Exec("DELETE FROM habit_logs WHERE habit_id IN (SELECT id FROM habits WHERE user_id = ?)", userID)
	if err != nil {
		return err
	}

	// Delete habits
	_, err = tx.Exec("DELETE FROM habits WHERE user_id = ?", userID)
	if err != nil {
		return err
	}

	// Delete goals (should cascade automatically, but let's be explicit)
	_, err = tx.Exec("DELETE FROM goals WHERE user_id = ?", userID)
	if err != nil {
		return err
	}

	// Delete roadmap ideas
	_, err = tx.Exec("DELETE FROM roadmap_ideas WHERE user_id = ?", userID)
	if err != nil {
		return err
	}

	// Delete roadmap likes
	_, err = tx.Exec("DELETE FROM roadmap_likes WHERE user_id = ?", userID)
	if err != nil {
		return err
	}

	// Delete user
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

// ResetUserData deletes all habits and their associated logs for a user
func ResetUserData(db *sql.DB, userID int64) error {
	// Start a transaction to ensure data consistency
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback if anything fails

	// Delete all habits (habit_logs will be deleted automatically due to ON DELETE CASCADE)
	_, err = tx.Exec(`DELETE FROM habits WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

// AdminUpdateUserPassword updates a user's password without requiring current password (admin only)
func AdminUpdateUserPassword(db *sql.DB, userID int64, newPassword string) error {
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, userID)
	return err
}

// ResetToken represents a password reset token
type ResetToken struct {
	Token     string
	UserID    int64
	Email     string
	Username  string
	Expiry    time.Time
	Used      bool
	CreatedAt time.Time
}

// InvalidateExistingTokens invalidates any existing reset tokens for a user
func InvalidateExistingTokens(db *sql.DB, email string) error {
	_, err := db.Exec(`
		DELETE FROM password_reset_tokens 
		WHERE email = ? AND used = FALSE
	`, email)
	return err
}

// CreateResetToken creates a new password reset token
func CreateResetToken(db *sql.DB, userID int64, email string, token string, expiry time.Time) error {
	_, err := db.Exec(`
		INSERT INTO password_reset_tokens (token, user_id, email, expiry)
		VALUES (?, ?, ?, ?)
	`, token, userID, email, expiry)
	return err
}

// GetResetToken retrieves a reset token
func GetResetToken(db *sql.DB, token string) (*ResetToken, error) {
	var rt ResetToken
	err := db.QueryRow(`
		SELECT t.token, t.user_id, t.email, t.expiry, t.used, t.created_at, u.first_name
		FROM password_reset_tokens t
		JOIN users u ON t.user_id = u.id
		WHERE t.token = ?
	`, token).Scan(&rt.Token, &rt.UserID, &rt.Email, &rt.Expiry, &rt.Used, &rt.CreatedAt, &rt.Username)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// MarkTokenUsed marks a reset token as used
func MarkTokenUsed(db *sql.DB, token string) error {
	_, err := db.Exec(`
		UPDATE password_reset_tokens
		SET used = TRUE
		WHERE token = ?
	`, token)
	return err
}

// UpdateUserPassword updates a user's password
func UpdateUserPassword(db *sql.DB, userID int64, hashedPassword string) error {
	_, err := db.Exec(`
		UPDATE users
		SET password_hash = ?
		WHERE id = ?
	`, hashedPassword, userID)
	return err
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// GetUsersWithNotificationsEnabled retrieves all users who have notifications enabled
func GetUsersWithNotificationsEnabled(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`
		SELECT id, first_name, last_name, email, show_confetti, show_weekdays, created_at, is_admin, notification_enabled
		FROM users
		WHERE notification_enabled = true
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
			&user.ShowConfetti,
			&user.ShowWeekdays,
			&user.CreatedAt,
			&user.IsAdmin,
			&user.NotificationEnabled,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetUsersWithHabitsAndNotificationsEnabled retrieves all users who have habits and notifications enabled
func GetUsersWithHabitsAndNotificationsEnabled(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`
		SELECT DISTINCT u.id, u.first_name, u.last_name, u.email, u.show_confetti, u.show_weekdays, u.created_at, u.is_admin, u.notification_enabled
		FROM users u
		JOIN habits h ON u.id = h.user_id
		WHERE u.notification_enabled = true
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
			&user.ShowConfetti,
			&user.ShowWeekdays,
			&user.CreatedAt,
			&user.IsAdmin,
			&user.NotificationEnabled,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetUsersWithNoHabitsAndNotificationsEnabled retrieves all users who have no habits but have notifications enabled
func GetUsersWithNoHabitsAndNotificationsEnabled(db *sql.DB) ([]*User, error) {
	rows, err := db.Query(`
		SELECT u.id, u.first_name, u.last_name, u.email, u.show_confetti, u.show_weekdays, u.created_at, u.is_admin, u.notification_enabled
		FROM users u
		LEFT JOIN habits h ON u.id = h.user_id
		WHERE h.id IS NULL AND u.notification_enabled = true
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
			&user.ShowConfetti,
			&user.ShowWeekdays,
			&user.CreatedAt,
			&user.IsAdmin,
			&user.NotificationEnabled,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// UpdateNotificationPreference updates a user's notification preference
func UpdateNotificationPreference(db *sql.DB, userID int64, enabled bool) error {
	_, err := db.Exec(`
		UPDATE users
		SET notification_enabled = ?
		WHERE id = ?
	`, enabled, userID)
	return err
}
