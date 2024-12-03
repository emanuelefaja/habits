package models

import (
	"database/sql"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUserByID retrieves a user from the database by their ID
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, first_name, last_name, email, created_at 
		FROM users 
		WHERE id = ?
	`, id).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user from the database by their email
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, first_name, last_name, email, created_at 
		FROM users 
		WHERE email = ?
	`, email).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// Create inserts a new user into the database
func (u *User) Create(db *sql.DB, passwordHash string) error {
	log.Println("Attempting to create user:", u.Email)

	result, err := db.Exec(`
		INSERT INTO users (first_name, last_name, email, password_hash, created_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, u.FirstName, u.LastName, u.Email, passwordHash)

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
		SET first_name = ?, last_name = ?, email = ? 
		WHERE id = ?
	`, u.FirstName, u.LastName, u.Email, u.ID)

	return err
}

// Delete removes a user from the database
func (u *User) Delete(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", u.ID)
	return err
}

// ValidatePassword checks if the provided password matches the stored hash
func ValidatePassword(db *sql.DB, email, password string) (bool, error) {
	var storedHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE email = ?", email).Scan(&storedHash)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil, nil
}

// InitializeDB creates the users table if it doesn't exist
func InitializeDB(db *sql.DB) error {
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
	return err
}
