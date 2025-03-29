package models

import (
	"database/sql"
	"mad/models/email"
	"os"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// MockEmailService implements the email.EmailService interface for testing
type MockEmailService struct {
	sentEmails map[string]bool
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sentEmails: make(map[string]bool),
	}
}

func (m *MockEmailService) SendTypedEmail(to string, template email.EmailTemplate, data interface{}) error {
	m.sentEmails[to+"-"+template.Name] = true
	return nil
}

func (m *MockEmailService) SendPasswordResetEmail(to, resetLink string, expiry time.Time) error {
	m.sentEmails[to+"-reset"] = true
	return nil
}

func (m *MockEmailService) SendPasswordResetSuccessEmail(to, username string) error {
	m.sentEmails[to+"-reset-success"] = true
	return nil
}

func (m *MockEmailService) SendReminderEmail(to string, firstName string, habits []email.HabitInfo, quote email.QuoteInfo) error {
	m.sentEmails[to+"-reminder"] = true
	return nil
}

func (m *MockEmailService) SendFirstHabitEmail(to string, firstName string, quote email.QuoteInfo) error {
	m.sentEmails[to+"-first-habit"] = true
	return nil
}

func (m *MockEmailService) SendSimpleEmail(to, subject, content string) error {
	m.sentEmails[to+"-"+subject] = true
	return nil
}

func (m *MockEmailService) GetCampaignManager() *email.CampaignManager {
	return email.NewCampaignManager(nil, m)
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Enable foreign keys for SQLite (this is critical for CASCADE to work)
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Initialize the database schema
	if err := InitDB(db); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return db
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *sql.DB) *User {
	user := &User{
		FirstName:           "Test",
		LastName:            "User",
		Email:               "test@example.com",
		ShowConfetti:        true,
		ShowWeekdays:        false,
		NotificationEnabled: true,
	}

	passHash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if err := user.Create(db, passHash); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// createTestHabit creates a test habit for a user
func createTestHabit(t *testing.T, db *sql.DB, userID int64) int64 {
	result, err := db.Exec(`
		INSERT INTO habits (user_id, name, emoji, habit_type, is_default, display_order)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "Test Habit", "✅", "binary", 0, 0)
	if err != nil {
		t.Fatalf("Failed to create test habit: %v", err)
	}

	habitID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get habit ID: %v", err)
	}

	return habitID
}

// TestGetUserByID tests the GetUserByID function
func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test user
	testUser := createTestUser(t, db)

	// Test successful retrieval
	user, err := GetUserByID(db, testUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if user.ID != testUser.ID {
		t.Errorf("Expected user ID %d, got %d", testUser.ID, user.ID)
	}
	if user.Email != testUser.Email {
		t.Errorf("Expected email %s, got %s", testUser.Email, user.Email)
	}
	if user.FirstName != testUser.FirstName {
		t.Errorf("Expected first name %s, got %s", testUser.FirstName, user.FirstName)
	}

	// Test non-existent user
	_, err = GetUserByID(db, 9999)
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}
}

// TestGetUserByEmail tests the GetUserByEmail function
func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test user
	testUser := createTestUser(t, db)

	// Test successful retrieval
	user, err := GetUserByEmail(db, testUser.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if user.ID != testUser.ID {
		t.Errorf("Expected user ID %d, got %d", testUser.ID, user.ID)
	}
	if user.Email != testUser.Email {
		t.Errorf("Expected email %s, got %s", testUser.Email, user.Email)
	}

	// Test case insensitivity
	user, err = GetUserByEmail(db, "TEST@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed with uppercase email: %v", err)
	}
	if user.ID != testUser.ID {
		t.Errorf("Case insensitive email lookup failed")
	}

	// Test non-existent user
	_, err = GetUserByEmail(db, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent email, got nil")
	}
}

// TestUserCreate tests the User.Create method
func TestUserCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test successful creation
	user := &User{
		FirstName:           "New",
		LastName:            "User",
		Email:               "new@example.com",
		ShowConfetti:        true,
		ShowWeekdays:        false,
		NotificationEnabled: true,
	}

	passHash, _ := HashPassword("password123")
	err := user.Create(db, passHash)
	if err != nil {
		t.Fatalf("User.Create failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}

	// Test duplicate email error
	duplicateUser := &User{
		FirstName:           "Duplicate",
		LastName:            "User",
		Email:               "new@example.com", // Same email
		ShowConfetti:        true,
		ShowWeekdays:        false,
		NotificationEnabled: true,
	}

	err = duplicateUser.Create(db, passHash)
	if err == nil {
		t.Error("Expected error for duplicate email, got nil")
	}

	// Check if error is a constraint violation
	sqliteErr, ok := err.(sqlite3.Error)
	if !ok || sqliteErr.ExtendedCode != sqlite3.ErrConstraintUnique {
		t.Errorf("Expected unique constraint error, got: %v", err)
	}
}

// TestUserUpdate tests the User.Update method
func TestUserUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := createTestUser(t, db)

	// Update user details
	user.FirstName = "Updated"
	user.LastName = "Name"
	user.Email = "updated@example.com"
	user.ShowConfetti = false
	user.NotificationEnabled = false

	if err := user.Update(db); err != nil {
		t.Fatalf("User.Update failed: %v", err)
	}

	// Retrieve updated user
	updatedUser, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if updatedUser.FirstName != "Updated" {
		t.Errorf("Expected first name 'Updated', got '%s'", updatedUser.FirstName)
	}
	if updatedUser.LastName != "Name" {
		t.Errorf("Expected last name 'Name', got '%s'", updatedUser.LastName)
	}
	if updatedUser.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got '%s'", updatedUser.Email)
	}
	if updatedUser.ShowConfetti != false {
		t.Errorf("Expected ShowConfetti to be false")
	}
	if updatedUser.NotificationEnabled != false {
		t.Errorf("Expected NotificationEnabled to be false")
	}
}

// TestUserDelete tests the User.Delete method
func TestUserDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := createTestUser(t, db)

	// Delete the user
	if err := user.Delete(db); err != nil {
		t.Fatalf("User.Delete failed: %v", err)
	}

	// Try to retrieve the deleted user
	_, err := GetUserByID(db, user.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted user, got nil")
	}
}

// TestValidatePassword tests the ValidatePassword function
func TestValidatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a user with a known password
	user := &User{
		FirstName:           "Password",
		LastName:            "Test",
		Email:               "password@example.com",
		ShowConfetti:        true,
		ShowWeekdays:        false,
		NotificationEnabled: true,
	}

	passHash, _ := HashPassword("correctpassword")
	err := user.Create(db, passHash)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test correct password
	valid, err := ValidatePassword(db, "password@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if !valid {
		t.Error("Expected valid password to return true")
	}

	// Test incorrect password
	valid, err = ValidatePassword(db, "password@example.com", "wrongpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if valid {
		t.Error("Expected invalid password to return false")
	}

	// Test non-existent user
	_, err = ValidatePassword(db, "nonexistent@example.com", "anypassword")
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}
}

// TestUpdatePassword tests the UpdatePassword function
func TestUpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a user with a known password
	user := createTestUser(t, db)

	// Set initial password
	initialPassHash, _ := HashPassword("initialpassword")
	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", initialPassHash, user.ID)
	if err != nil {
		t.Fatalf("Failed to set initial password: %v", err)
	}

	// Update password
	err = UpdatePassword(db, user.ID, "initialpassword", "newpassword")
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	// Verify new password works
	valid, err := ValidatePassword(db, user.Email, "newpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if !valid {
		t.Error("Expected new password to be valid")
	}

	// Verify old password doesn't work
	valid, err = ValidatePassword(db, user.Email, "initialpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if valid {
		t.Error("Expected old password to be invalid")
	}

	// Test with incorrect current password
	err = UpdatePassword(db, user.ID, "wrongpassword", "anotherpassword")
	if err == nil {
		t.Error("Expected error when providing incorrect current password")
	}
}

// TestDeleteUserAndData tests the DeleteUserAndData function
func TestDeleteUserAndData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Set required environment variables for email service
	os.Setenv("SMTP_HOST", "localhost")
	os.Setenv("SMTP_USERNAME", "test")
	os.Setenv("SMTP_PASSWORD", "test")
	os.Setenv("SMTP_FROM_NAME", "Test")
	os.Setenv("SMTP_FROM_EMAIL", "test@example.com")

	// Create a test user with a habit and log
	user := createTestUser(t, db)
	habitID := createTestHabit(t, db, user.ID)

	// Create a habit log
	_, err := db.Exec(`
		INSERT INTO habit_logs (habit_id, date, status, value)
		VALUES (?, ?, ?, ?)
	`, habitID, time.Now().Format("2006-01-02"), "done", "1")
	if err != nil {
		t.Fatalf("Failed to create habit log: %v", err)
	}

	// Create roadmap ideas
	_, err = db.Exec(`
		INSERT INTO roadmap_ideas (user_id, idea_text)
		VALUES (?, ?)
	`, user.ID, "Test idea")
	if err != nil {
		t.Fatalf("Failed to create roadmap idea: %v", err)
	}

	// Create roadmap likes
	_, err = db.Exec(`
		INSERT INTO roadmap_likes (user_id, card_id)
		VALUES (?, ?)
	`, user.ID, "card1")
	if err != nil {
		t.Fatalf("Failed to create roadmap like: %v", err)
	}

	// Delete user and all associated data
	err = DeleteUserAndData(db, user.ID)
	if err != nil {
		t.Fatalf("DeleteUserAndData failed: %v", err)
	}

	// Verify user is deleted
	_, err = GetUserByID(db, user.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted user")
	}

	// Verify habits are deleted
	var habitCount int
	err = db.QueryRow("SELECT COUNT(*) FROM habits WHERE user_id = ?", user.ID).Scan(&habitCount)
	if err != nil {
		t.Fatalf("Failed to count habits: %v", err)
	}
	if habitCount > 0 {
		t.Errorf("Expected 0 habits, got %d", habitCount)
	}

	// Verify habit logs are deleted
	var logCount int
	err = db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = ?", habitID).Scan(&logCount)
	if err != nil {
		t.Fatalf("Failed to count habit logs: %v", err)
	}
	if logCount > 0 {
		t.Errorf("Expected 0 habit logs, got %d", logCount)
	}

	// Verify roadmap ideas are deleted
	var ideaCount int
	err = db.QueryRow("SELECT COUNT(*) FROM roadmap_ideas WHERE user_id = ?", user.ID).Scan(&ideaCount)
	if err != nil {
		t.Fatalf("Failed to count roadmap ideas: %v", err)
	}
	if ideaCount > 0 {
		t.Errorf("Expected 0 ideas, got %d", ideaCount)
	}

	// Verify roadmap likes are deleted
	var likeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM roadmap_likes WHERE user_id = ?", user.ID).Scan(&likeCount)
	if err != nil {
		t.Fatalf("Failed to count roadmap likes: %v", err)
	}
	if likeCount > 0 {
		t.Errorf("Expected 0 likes, got %d", likeCount)
	}
}

// TestResetUserData tests the ResetUserData function
func TestResetUserData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test user with a habit and log
	user := createTestUser(t, db)
	habitID := createTestHabit(t, db, user.ID)

	// Create habit logs
	_, err := db.Exec(`
		INSERT INTO habit_logs (habit_id, date, status, value)
		VALUES (?, ?, ?, ?)
	`, habitID, time.Now().Format("2006-01-02"), "done", "1")
	if err != nil {
		t.Fatalf("Failed to create habit log: %v", err)
	}

	// Reset user data
	err = ResetUserData(db, user.ID)
	if err != nil {
		t.Fatalf("ResetUserData failed: %v", err)
	}

	// Verify user still exists
	_, err = GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("User should still exist after reset: %v", err)
	}

	// Verify habits are deleted
	var habitCount int
	err = db.QueryRow("SELECT COUNT(*) FROM habits WHERE user_id = ?", user.ID).Scan(&habitCount)
	if err != nil {
		t.Fatalf("Failed to count habits: %v", err)
	}
	if habitCount > 0 {
		t.Errorf("Expected 0 habits, got %d", habitCount)
	}

	// Verify habit logs are deleted (due to ON DELETE CASCADE)
	var logCount int
	err = db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = ?", habitID).Scan(&logCount)
	if err != nil {
		t.Fatalf("Failed to count habit logs: %v", err)
	}
	if logCount > 0 {
		t.Errorf("Expected 0 habit logs, got %d", logCount)
	}
}

// TestPasswordResetFlow tests the complete password reset flow
func TestPasswordResetFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test user
	user := createTestUser(t, db)

	// 1. Invalidate any existing tokens
	err := InvalidateExistingTokens(db, user.Email)
	if err != nil {
		t.Fatalf("InvalidateExistingTokens failed: %v", err)
	}

	// 2. Create a reset token
	token := "test-token-123"
	expiry := time.Now().Add(24 * time.Hour)
	err = CreateResetToken(db, user.ID, user.Email, token, expiry)
	if err != nil {
		t.Fatalf("CreateResetToken failed: %v", err)
	}

	// 3. Retrieve the token
	resetToken, err := GetResetToken(db, token)
	if err != nil {
		t.Fatalf("GetResetToken failed: %v", err)
	}

	if resetToken.Token != token {
		t.Errorf("Expected token %s, got %s", token, resetToken.Token)
	}
	if resetToken.UserID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, resetToken.UserID)
	}
	if resetToken.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, resetToken.Email)
	}
	if resetToken.Used {
		t.Error("Expected token to be unused")
	}

	// 4. Mark token as used
	err = MarkTokenUsed(db, token)
	if err != nil {
		t.Fatalf("MarkTokenUsed failed: %v", err)
	}

	// 5. Verify token is marked as used
	resetToken, err = GetResetToken(db, token)
	if err != nil {
		t.Fatalf("GetResetToken failed after marking used: %v", err)
	}
	if !resetToken.Used {
		t.Error("Expected token to be marked as used")
	}

	// 6. Update password with new hash
	newHash, _ := HashPassword("newpassword")
	err = UpdateUserPassword(db, user.ID, newHash)
	if err != nil {
		t.Fatalf("UpdateUserPassword failed: %v", err)
	}

	// 7. Verify new password works
	valid, err := ValidatePassword(db, user.Email, "newpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if !valid {
		t.Error("Expected new password to be valid")
	}
}

// TestHashPassword tests the HashPassword function
func TestHashPassword(t *testing.T) {
	// Test password hashing
	hash, err := HashPassword("testpassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Verify hash is not empty
	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify hash is not the same as the original password
	if hash == "testpassword" {
		t.Error("Hash should not be the same as the password")
	}

	// Verify hash is a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("testpassword"))
	if err != nil {
		t.Errorf("Hash validation failed: %v", err)
	}

	// Verify hash fails with wrong password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpassword"))
	if err == nil {
		t.Error("Expected error when comparing hash with wrong password")
	}
}

// TestNotificationPreferences tests the notification-related functions
func TestNotificationPreferences(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test users
	user1 := createTestUser(t, db)
	user2 := &User{
		FirstName:           "User",
		LastName:            "Two",
		Email:               "user2@example.com",
		ShowConfetti:        true,
		ShowWeekdays:        false,
		NotificationEnabled: false,
	}
	passHash, _ := HashPassword("password123")
	user2.Create(db, passHash)

	// Create a habit for user1
	createTestHabit(t, db, user1.ID)

	// 1. Test UpdateNotificationPreference
	err := UpdateNotificationPreference(db, user1.ID, false)
	if err != nil {
		t.Fatalf("UpdateNotificationPreference failed: %v", err)
	}

	updatedUser, err := GetUserByID(db, user1.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if updatedUser.NotificationEnabled {
		t.Error("Expected NotificationEnabled to be false after update")
	}

	// 2. Test GetUsersWithNotificationsEnabled
	err = UpdateNotificationPreference(db, user2.ID, true)
	if err != nil {
		t.Fatalf("UpdateNotificationPreference failed: %v", err)
	}

	usersWithNotifications, err := GetUsersWithNotificationsEnabled(db)
	if err != nil {
		t.Fatalf("GetUsersWithNotificationsEnabled failed: %v", err)
	}
	if len(usersWithNotifications) != 1 {
		t.Errorf("Expected 1 user with notifications enabled, got %d", len(usersWithNotifications))
	}

	// 3. Test GetUsersWithHabitsAndNotificationsEnabled
	err = UpdateNotificationPreference(db, user1.ID, true)
	if err != nil {
		t.Fatalf("UpdateNotificationPreference failed: %v", err)
	}

	usersWithHabitsAndNotifications, err := GetUsersWithHabitsAndNotificationsEnabled(db)
	if err != nil {
		t.Fatalf("GetUsersWithHabitsAndNotificationsEnabled failed: %v", err)
	}
	if len(usersWithHabitsAndNotifications) != 1 {
		t.Errorf("Expected 1 user with habits and notifications, got %d", len(usersWithHabitsAndNotifications))
	}
	if usersWithHabitsAndNotifications[0].ID != user1.ID {
		t.Errorf("Expected user1, got user with ID %d", usersWithHabitsAndNotifications[0].ID)
	}

	// 4. Test GetUsersWithNoHabitsAndNotificationsEnabled
	usersWithNoHabitsAndNotifications, err := GetUsersWithNoHabitsAndNotificationsEnabled(db)
	if err != nil {
		t.Fatalf("GetUsersWithNoHabitsAndNotificationsEnabled failed: %v", err)
	}
	if len(usersWithNoHabitsAndNotifications) != 1 {
		t.Errorf("Expected 1 user with no habits and notifications, got %d", len(usersWithNoHabitsAndNotifications))
	}
	if usersWithNoHabitsAndNotifications[0].ID != user2.ID {
		t.Errorf("Expected user2, got user with ID %d", usersWithNoHabitsAndNotifications[0].ID)
	}
}

// TestAdminUpdateUserPassword tests the AdminUpdateUserPassword function
func TestAdminUpdateUserPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a test user
	user := createTestUser(t, db)

	// Admin update password
	err := AdminUpdateUserPassword(db, user.ID, "adminsetpassword")
	if err != nil {
		t.Fatalf("AdminUpdateUserPassword failed: %v", err)
	}

	// Verify new password works
	valid, err := ValidatePassword(db, user.Email, "adminsetpassword")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if !valid {
		t.Error("Expected admin-set password to be valid")
	}

	// Verify old password doesn't work
	valid, err = ValidatePassword(db, user.Email, "password123")
	if err != nil {
		t.Fatalf("ValidatePassword failed: %v", err)
	}
	if valid {
		t.Error("Expected old password to be invalid after admin update")
	}
}
