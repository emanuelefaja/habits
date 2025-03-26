// Package campaigns contains integration tests for email campaign functionality
package campaigns

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"mad/models"
	"mad/models/email"
)

// TestUserDeletion tests that when a user is deleted, their email subscriptions are properly anonymized
func TestUserDeletion(t *testing.T) {
	// Load environment variables from parent directory
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Error loading .env file from parent directory: %v", err)
	}

	// Open database connection
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "../../habits.db"
	}

	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Verify DB connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Error connecting to database: %v", err)
	}

	// Create a test user for deletion
	testUser, err := createTestUserForDeletion(db)
	if err != nil {
		t.Fatalf("Error creating test user: %v", err)
	}
	t.Logf("Created test user with ID: %d and email: %s", testUser.ID, testUser.Email)

	// Subscribe the user to a campaign
	err = subscribeUserToCampaignForDeletion(db, testUser)
	if err != nil {
		t.Fatalf("Error subscribing user to campaign: %v", err)
	}
	t.Logf("Successfully subscribed user to campaign")

	// Print subscription status before deletion
	printSubscriptionStatusForDeletion(t, db, testUser.ID)

	// Delete the user
	err = models.DeleteUserAndData(db, testUser.ID)
	if err != nil {
		t.Fatalf("Error deleting user: %v", err)
	}
	t.Logf("Successfully deleted user with ID: %d", testUser.ID)

	// Verify user deletion
	_, err = models.GetUserByID(db, testUser.ID)
	if err == nil {
		t.Fatal("ERROR: User still exists in database!")
	} else {
		t.Logf("Verified user deletion - user no longer exists in database")
	}

	// Check if subscriptions were properly anonymized
	checkSubscriptionsAfterDeletion(t, db, testUser.Email)

	t.Logf("Test completed successfully!")
}

// createTestUserForDeletion creates a new user for testing
func createTestUserForDeletion(db *sql.DB) (*models.User, error) {
	// Generate test email with timestamp to ensure uniqueness
	email := fmt.Sprintf("testuser_%d@example.com", os.Getpid())
	user := &models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
	}

	// Generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create the user
	err = user.Create(db, string(passwordHash))
	if err != nil {
		return nil, err
	}

	// Get the user to retrieve the assigned ID
	createdUser, err := models.GetUserByEmail(db, email)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// subscribeUserToCampaignForDeletion subscribes the test user to a campaign
func subscribeUserToCampaignForDeletion(db *sql.DB, user *models.User) error {
	// Initialize email service
	emailService, err := email.NewSMTPEmailService(email.SMTPConfig{
		Host:        os.Getenv("SMTP_HOST"),
		Port:        587,
		Username:    os.Getenv("SMTP_USERNAME"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		FromName:    os.Getenv("SMTP_FROM_NAME"),
		FromEmail:   os.Getenv("SMTP_FROM_EMAIL"),
		TemplateDir: "../../ui/email",
	})
	if err != nil {
		return fmt.Errorf("could not initialize email service: %w", err)
	}

	// Create campaign manager
	campaignManager := email.NewCampaignManager(db, emailService)

	// Subscribe user to the digital-detox campaign
	err = campaignManager.SubscribeUser(user.Email, "digital-detox", int(user.ID))
	if err != nil {
		return fmt.Errorf("could not subscribe user: %w", err)
	}

	return nil
}

// printSubscriptionStatusForDeletion prints the current subscription status
func printSubscriptionStatusForDeletion(t *testing.T, db *sql.DB, userID int64) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM email_subscriptions WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		t.Logf("Error checking subscriptions: %v", err)
		return
	}

	t.Logf("User has %d active subscriptions before deletion", count)

	rows, err := db.Query(`
		SELECT id, email, campaign_id, status
		FROM email_subscriptions 
		WHERE user_id = ?`,
		userID)
	if err != nil {
		t.Logf("Error querying subscriptions: %v", err)
		return
	}
	defer rows.Close()

	t.Log("Current subscriptions:")
	t.Log("ID | Email | Campaign | Status")
	t.Log("--------------------------------")
	for rows.Next() {
		var id int
		var email, campaignID, status string
		if err := rows.Scan(&id, &email, &campaignID, &status); err != nil {
			t.Logf("Error scanning subscription: %v", err)
			continue
		}
		t.Logf("%d | %s | %s | %s", id, email, campaignID, status)
	}
	t.Log("--------------------------------")
}

// checkSubscriptionsAfterDeletion checks the status of subscriptions after user deletion
func checkSubscriptionsAfterDeletion(t *testing.T, db *sql.DB, userEmail string) {
	// Get subscriptions that match the original email
	rows, err := db.Query(`
		SELECT id, email, campaign_id, status, user_id
		FROM email_subscriptions 
		WHERE email = ?`,
		userEmail)
	if err != nil {
		t.Logf("Error querying subscriptions by email: %v", err)
		return
	}
	defer rows.Close()

	originalEmailSubscriptions := 0
	for rows.Next() {
		originalEmailSubscriptions++
		var id int
		var email, campaignID, status string
		var userID sql.NullInt64
		if err := rows.Scan(&id, &email, &campaignID, &status, &userID); err != nil {
			t.Logf("Error scanning subscription: %v", err)
			continue
		}
		t.Logf("WARNING: Found subscription with original email: ID=%d, Email=%s, Campaign=%s, Status=%s, UserID=%v",
			id, email, campaignID, status, userID)
	}

	if originalEmailSubscriptions > 0 {
		t.Errorf("Test FAILED: Found %d subscriptions with the original email", originalEmailSubscriptions)
	} else {
		t.Logf("Test PASSED: No subscriptions found with the original email")
	}

	// Check for anonymized subscriptions
	rows, err = db.Query(`
		SELECT id, email, campaign_id, status
		FROM email_subscriptions 
		WHERE email LIKE 'deleted-user-%@anonymous.com'`)
	if err != nil {
		t.Logf("Error querying anonymized subscriptions: %v", err)
		return
	}
	defer rows.Close()

	anonymizedSubscriptions := 0
	t.Log("Anonymized subscriptions:")
	t.Log("ID | Email | Campaign | Status")
	t.Log("--------------------------------")
	for rows.Next() {
		anonymizedSubscriptions++
		var id int
		var email, campaignID, status string
		if err := rows.Scan(&id, &email, &campaignID, &status); err != nil {
			t.Logf("Error scanning subscription: %v", err)
			continue
		}
		t.Logf("%d | %s | %s | %s", id, email, campaignID, status)
	}
	t.Log("--------------------------------")

	t.Logf("Found %d anonymized subscriptions", anonymizedSubscriptions)

	if anonymizedSubscriptions == 0 {
		t.Error("Test FAILED: No anonymized subscriptions found after user deletion")
	}
}
