// Package campaigns contains integration tests for email campaign functionality
package campaigns

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"mad/models"
	"mad/models/email"
)

// EmailSubscriptionWithToken represents a subscription with its token for testing
type EmailSubscriptionWithToken struct {
	ID             int
	UserID         sql.NullInt64
	Email          string
	CampaignID     string
	Token          string
	SubscribedAt   time.Time
	Status         string
	LastEmailSent  int
	UnsubscribedAt sql.NullTime
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TestUnsubscribeFlow tests the unsubscribe flow including link generation and token validation
func TestUnsubscribeFlow(t *testing.T) {
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

	// Initialize email service for campaign manager
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
		t.Fatalf("Error initializing email service: %v", err)
	}

	// Create campaign manager
	campaignManager := email.NewCampaignManager(db, emailService)

	// ***** TEST 1: Generate Unsubscribe Link *****
	t.Run("GenerateUnsubscribeLink", func(t *testing.T) {
		// Create a test user
		testUser, err := createTestUserForUnsubscribe(db)
		if err != nil {
			t.Fatalf("Error creating test user: %v", err)
		}
		t.Logf("Created test user with ID: %d and email: %s", testUser.ID, testUser.Email)

		// Subscribe user to a campaign
		err = campaignManager.SubscribeUser(testUser.Email, "digital-detox", int(testUser.ID))
		if err != nil {
			t.Fatalf("Error subscribing user: %v", err)
		}
		t.Logf("Successfully subscribed user to campaign")

		// Get the subscription to retrieve the token
		sub, err := getSubscriptionWithUnsubscribeToken(db, testUser.Email, "digital-detox")
		if err != nil {
			t.Fatalf("Error getting subscription: %v", err)
		}

		printUnsubscribeSubscription(t, sub)

		// Verify token exists
		if sub.Token == "" {
			t.Fatal("ERROR: No token generated for subscription")
		}

		// Generate unsubscribe link
		unsubscribeLink := email.GenerateUnsubscribeLink(testUser.Email, "digital-detox", sub.Token)
		t.Logf("Generated unsubscribe link: %s", unsubscribeLink)

		// Verify link format
		parsedURL, err := url.Parse(unsubscribeLink)
		if err != nil {
			t.Fatalf("ERROR: Invalid URL generated: %v", err)
		}

		queryParams := parsedURL.Query()
		if queryParams.Get("email") != testUser.Email {
			t.Fatalf("ERROR: Email in link doesn't match. Expected: %s, Got: %s",
				testUser.Email, queryParams.Get("email"))
		}

		if queryParams.Get("campaign") != "digital-detox" {
			t.Fatalf("ERROR: Campaign ID in link doesn't match. Expected: %s, Got: %s",
				"digital-detox", queryParams.Get("campaign"))
		}

		if queryParams.Get("token") != sub.Token {
			t.Fatalf("ERROR: Token in link doesn't match. Expected: %s, Got: %s",
				sub.Token, queryParams.Get("token"))
		}

		t.Logf("PASS: Unsubscribe link format is valid")

		// ***** TEST 2: Validate Unsubscribe Token *****
		t.Run("ValidateToken", func(t *testing.T) {
			// Test invalid token
			invalidToken := "invalid-token-123"
			isValid := validateUnsubscribeToken(db, testUser.Email, "digital-detox", invalidToken)
			if isValid {
				t.Fatal("ERROR: Invalid token was validated as valid")
			}
			t.Logf("PASS: Invalid token correctly rejected")

			// Test valid token
			isValid = validateUnsubscribeToken(db, testUser.Email, "digital-detox", sub.Token)
			if !isValid {
				t.Fatal("ERROR: Valid token was rejected")
			}
			t.Logf("PASS: Valid token correctly validated")

			// Test mismatched email
			isValid = validateUnsubscribeToken(db, "wrong@example.com", "digital-detox", sub.Token)
			if isValid {
				t.Fatal("ERROR: Token with wrong email was validated")
			}
			t.Logf("PASS: Token with wrong email correctly rejected")

			// Test mismatched campaign
			isValid = validateUnsubscribeToken(db, testUser.Email, "wrong-campaign", sub.Token)
			if isValid {
				t.Fatal("ERROR: Token with wrong campaign was validated")
			}
			t.Logf("PASS: Token with wrong campaign correctly rejected")
		})

		// ***** TEST 3: Process Unsubscribe *****
		t.Run("ProcessUnsubscribe", func(t *testing.T) {
			// Process unsubscribe (simulating clicking the link and submitting form)
			err = processUnsubscribeRequest(db, testUser.Email, "digital-detox", sub.Token)
			if err != nil {
				t.Fatalf("ERROR: Failed to process unsubscribe: %v", err)
			}
			t.Logf("Processed unsubscribe request")

			// Verify subscription is now unsubscribed
			subAfterUnsub, err := getSubscriptionWithUnsubscribeToken(db, testUser.Email, "digital-detox")
			if err != nil {
				t.Fatalf("Error getting subscription after unsubscribe: %v", err)
			}

			printUnsubscribeSubscription(t, subAfterUnsub)

			if subAfterUnsub.Status != "unsubscribed" {
				t.Fatalf("ERROR: Subscription status after unsubscribe is %s, expected 'unsubscribed'",
					subAfterUnsub.Status)
			}

			if !subAfterUnsub.UnsubscribedAt.Valid {
				t.Fatal("ERROR: Unsubscribe timestamp not set")
			}

			t.Logf("PASS: Subscription successfully unsubscribed")

			// ***** TEST 4: Duplicate Unsubscribe *****
			t.Run("DuplicateUnsubscribe", func(t *testing.T) {
				// Process the same unsubscribe again
				err = processUnsubscribeRequest(db, testUser.Email, "digital-detox", sub.Token)
				if err != nil {
					t.Fatalf("ERROR: Failed to process duplicate unsubscribe: %v", err)
				}
				t.Logf("Processed duplicate unsubscribe request")

				// Should still be unsubscribed
				subAfterDuplicate, err := getSubscriptionWithUnsubscribeToken(db, testUser.Email, "digital-detox")
				if err != nil {
					t.Fatalf("Error getting subscription after duplicate unsubscribe: %v", err)
				}

				if subAfterDuplicate.Status != "unsubscribed" {
					t.Fatalf("ERROR: Subscription status changed after duplicate unsubscribe to %s",
						subAfterDuplicate.Status)
				}

				t.Logf("PASS: Duplicate unsubscribe request handled correctly")
			})
		})

		// Clean up test data
		t.Logf("Cleaning up test data")

		// Delete the test user (cascades to subscriptions)
		err = models.DeleteUserAndData(db, testUser.ID)
		if err != nil {
			t.Fatalf("Error deleting test user: %v", err)
		}
		t.Logf("Deleted test user with ID: %d", testUser.ID)

		t.Logf("All unsubscribe flow tests PASSED!")
	})
}

// createTestUserForUnsubscribe creates a new user for testing
func createTestUserForUnsubscribe(db *sql.DB) (*models.User, error) {
	// Generate test email with timestamp to ensure uniqueness
	email := fmt.Sprintf("testuser_unsub_%d@example.com", time.Now().UnixNano())
	user := &models.User{
		FirstName: "Test",
		LastName:  "Unsubscribe",
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
		return nil, fmt.Errorf("error executing insert: %w", err)
	}

	// Get the user to retrieve the assigned ID
	createdUser, err := models.GetUserByEmail(db, email)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// getSubscriptionWithUnsubscribeToken retrieves subscription data including the token
func getSubscriptionWithUnsubscribeToken(db *sql.DB, email, campaignID string) (*EmailSubscriptionWithToken, error) {
	var sub EmailSubscriptionWithToken
	err := db.QueryRow(`
		SELECT id, user_id, email, campaign_id, token, subscribed_at, 
		       status, last_email_sent, unsubscribed_at, created_at, updated_at
		FROM email_subscriptions 
		WHERE email = ? AND campaign_id = ?
		LIMIT 1
	`, email, campaignID).Scan(
		&sub.ID, &sub.UserID, &sub.Email, &sub.CampaignID, &sub.Token, &sub.SubscribedAt,
		&sub.Status, &sub.LastEmailSent, &sub.UnsubscribedAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &sub, nil
}

// validateUnsubscribeToken checks if a token is valid for a given email and campaign
func validateUnsubscribeToken(db *sql.DB, email, campaignID, token string) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM email_subscriptions 
		WHERE email = ? AND campaign_id = ? AND token = ?
	`, email, campaignID, token).Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

// processUnsubscribeRequest simulates the unsubscribe action after clicking the link
func processUnsubscribeRequest(db *sql.DB, email, campaignID, token string) error {
	// First validate the token
	if !validateUnsubscribeToken(db, email, campaignID, token) {
		return fmt.Errorf("invalid unsubscribe token")
	}

	// Then update the subscription status
	_, err := db.Exec(`
		UPDATE email_subscriptions 
		SET status = 'unsubscribed', 
		    unsubscribed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE email = ? AND campaign_id = ? AND token = ?
	`, email, campaignID, token)

	return err
}

// printUnsubscribeSubscription prints the details of a subscription
func printUnsubscribeSubscription(t *testing.T, sub *EmailSubscriptionWithToken) {
	t.Log("Subscription details:")
	t.Log("--------------------")
	t.Logf("ID:             %d", sub.ID)
	t.Logf("Email:          %s", sub.Email)
	t.Logf("Campaign:       %s", sub.CampaignID)
	t.Logf("Token:          %s", sub.Token)
	t.Logf("UserID:         %v", sub.UserID)
	t.Logf("Status:         %s", sub.Status)
	t.Logf("LastEmailSent:  %d", sub.LastEmailSent)
	t.Logf("SubscribedAt:   %v", sub.SubscribedAt)
	t.Logf("UnsubscribedAt: %v", sub.UnsubscribedAt)
	t.Logf("CreatedAt:      %v", sub.CreatedAt)
	t.Logf("UpdatedAt:      %v", sub.UpdatedAt)
	t.Log("--------------------")
}
