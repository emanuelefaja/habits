// Package campaigns contains integration tests for email campaign functionality
package campaigns

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"mad/models"
	"mad/models/email"
)

// EmailSubscriptionInfo represents a subscription for testing
type EmailSubscriptionInfo struct {
	ID             int
	UserID         sql.NullInt64
	Email          string
	CampaignID     string
	SubscribedAt   time.Time
	Status         string
	LastEmailSent  int
	UnsubscribedAt sql.NullTime
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TestSubscriptionFlow tests the subscription flow for both registered and anonymous users
func TestSubscriptionFlow(t *testing.T) {
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

	// ***** TEST 1: Registered User Subscription *****
	t.Run("RegisteredUserSubscription", func(t *testing.T) {
		// Create a test user
		testUser, err := createTestUserForSubscription(db)
		if err != nil {
			t.Fatalf("Error creating test user: %v", err)
		}
		t.Logf("Created test user with ID: %d and email: %s", testUser.ID, testUser.Email)

		// Subscribe user to a campaign
		err = campaignManager.SubscribeUser(testUser.Email, "digital-detox", int(testUser.ID))
		if err != nil {
			t.Fatalf("Error subscribing user: %v", err)
		}
		t.Logf("Successfully subscribed registered user to campaign")

		// Verify subscription was stored correctly
		sub, err := getSubscriptionInfo(db, testUser.Email, "digital-detox")
		if err != nil {
			t.Fatalf("Error getting subscription: %v", err)
		}

		printSubscriptionInfo(t, sub)
		if !sub.UserID.Valid || int64(sub.UserID.Int64) != testUser.ID {
			t.Fatal("ERROR: User ID not correctly stored in subscription")
		}
		if sub.Status != "active" {
			t.Fatal("ERROR: Subscription status is not active")
		}

		// Try subscribing again (should be idempotent)
		t.Logf("Testing idempotency: subscribing same user again...")
		err = campaignManager.SubscribeUser(testUser.Email, "digital-detox", int(testUser.ID))
		if err != nil {
			t.Fatalf("Error re-subscribing user: %v", err)
		}
		t.Logf("Successfully re-subscribed user (idempotent operation)")

		// Check that we don't have duplicate subscriptions
		count := countSubscriptionsFor(db, testUser.Email, "digital-detox")
		if count != 1 {
			t.Fatalf("ERROR: Expected 1 subscription, got %d", count)
		}
		t.Logf("PASS: Only one subscription exists after multiple subscribe attempts")

		// Clean up at the end of this subtest
		defer func() {
			err = models.DeleteUserAndData(db, testUser.ID)
			if err != nil {
				t.Logf("Warning: Error deleting test user: %v", err)
			} else {
				t.Logf("Cleanup: Deleted test user with ID: %d", testUser.ID)
			}
		}()

		// ***** TEST 2: Anonymous User Subscription *****
		t.Run("AnonymousUserSubscription", func(t *testing.T) {
			// Generate a unique anonymous email
			anonEmail := fmt.Sprintf("anon_%d@example.com", time.Now().UnixNano())

			// Subscribe anonymous user
			err = campaignManager.SubscribeUser(anonEmail, "digital-detox", 0) // 0 = no user ID
			if err != nil {
				t.Fatalf("Error subscribing anonymous user: %v", err)
			}
			t.Logf("Successfully subscribed anonymous user to campaign")

			// Verify subscription was stored correctly
			anonSub, err := getSubscriptionInfo(db, anonEmail, "digital-detox")
			if err != nil {
				t.Fatalf("Error getting anonymous subscription: %v", err)
			}

			printSubscriptionInfo(t, anonSub)
			if anonSub.UserID.Valid {
				t.Fatal("ERROR: Anonymous subscription has a user ID when it shouldn't")
			}
			if anonSub.Status != "active" {
				t.Fatal("ERROR: Anonymous subscription status is not active")
			}

			// Try subscribing anonymous user again (should be idempotent)
			t.Logf("Testing idempotency: subscribing same anonymous user again...")
			err = campaignManager.SubscribeUser(anonEmail, "digital-detox", 0)
			if err != nil {
				t.Fatalf("Error re-subscribing anonymous user: %v", err)
			}
			t.Logf("Successfully re-subscribed anonymous user (idempotent operation)")

			// Check that we don't have duplicate subscriptions
			anonCount := countSubscriptionsFor(db, anonEmail, "digital-detox")
			if anonCount != 1 {
				t.Fatalf("ERROR: Expected 1 anonymous subscription, got %d", anonCount)
			}
			t.Logf("PASS: Only one subscription exists after multiple anonymous subscribe attempts")

			// Clean up anonymous subscription at the end
			defer func() {
				err = deleteSubscriptionFor(db, anonEmail)
				if err != nil {
					t.Logf("Warning: Error deleting anonymous subscription: %v", err)
				} else {
					t.Logf("Cleanup: Deleted anonymous subscription for: %s", anonEmail)
				}
			}()
		})

		// ***** TEST 3: Cross-Campaign Subscriptions *****
		t.Run("MultipleCampaignSubscriptions", func(t *testing.T) {
			// Check if onboarding campaign exists
			if _, err := email.GetCampaign("onboarding"); err != nil {
				t.Skipf("Onboarding campaign not available: %v", err)
			}

			// Subscribe to a second campaign
			err = campaignManager.SubscribeUser(testUser.Email, "onboarding", int(testUser.ID))
			if err != nil {
				t.Fatalf("Error subscribing to second campaign: %v", err)
			}
			t.Logf("Successfully subscribed user to second campaign")

			// Verify multiple subscriptions
			count := countUserSubscriptionsFor(db, testUser.ID)
			if count < 2 {
				t.Fatalf("ERROR: Expected at least 2 subscriptions for user, got %d", count)
			}
			t.Logf("PASS: User has %d subscriptions across campaigns", count)
		})

		// ***** TEST 4: Unsubscribe then Resubscribe *****
		t.Run("UnsubscribeAndResubscribe", func(t *testing.T) {
			// Unsubscribe the user
			err = campaignManager.UnsubscribeUser(testUser.Email, "digital-detox")
			if err != nil {
				t.Fatalf("Error unsubscribing user: %v", err)
			}
			t.Logf("Successfully unsubscribed user from campaign")

			// Verify unsubscribe worked
			sub, err = getSubscriptionInfo(db, testUser.Email, "digital-detox")
			if err != nil {
				t.Fatalf("Error getting subscription after unsubscribe: %v", err)
			}

			printSubscriptionInfo(t, sub)
			if sub.Status != "unsubscribed" {
				t.Fatal("ERROR: Subscription status is not unsubscribed after unsubscribe")
			}
			if !sub.UnsubscribedAt.Valid {
				t.Fatal("ERROR: UnsubscribedAt timestamp not set after unsubscribe")
			}

			// Resubscribe
			t.Logf("Testing resubscription after unsubscribe...")
			err = campaignManager.SubscribeUser(testUser.Email, "digital-detox", int(testUser.ID))
			if err != nil {
				t.Fatalf("Error resubscribing user after unsubscribe: %v", err)
			}
			t.Logf("Successfully resubscribed user after unsubscribe")

			// Verify resubscribe worked
			sub, err = getSubscriptionInfo(db, testUser.Email, "digital-detox")
			if err != nil {
				t.Fatalf("Error getting subscription after resubscribe: %v", err)
			}

			printSubscriptionInfo(t, sub)
			if sub.Status != "active" {
				t.Fatal("ERROR: Subscription status is not active after resubscribe")
			}
			if sub.UnsubscribedAt.Valid {
				t.Fatal("ERROR: UnsubscribedAt timestamp still set after resubscribe")
			}
		})
	})

	t.Logf("All subscription tests completed successfully!")
}

// createTestUserForSubscription creates a new user for testing
func createTestUserForSubscription(db *sql.DB) (*models.User, error) {
	// Generate test email with timestamp to ensure uniqueness
	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())

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
		return nil, fmt.Errorf("error executing insert: %w", err)
	}

	// Get the user to retrieve the assigned ID
	createdUser, err := models.GetUserByEmail(db, email)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// getSubscriptionInfo retrieves subscription data
func getSubscriptionInfo(db *sql.DB, email, campaignID string) (*EmailSubscriptionInfo, error) {
	var sub EmailSubscriptionInfo
	err := db.QueryRow(`
		SELECT id, user_id, email, campaign_id, subscribed_at, 
		       status, last_email_sent, unsubscribed_at, created_at, updated_at
		FROM email_subscriptions 
		WHERE email = ? AND campaign_id = ?
		LIMIT 1
	`, email, campaignID).Scan(
		&sub.ID, &sub.UserID, &sub.Email, &sub.CampaignID, &sub.SubscribedAt,
		&sub.Status, &sub.LastEmailSent, &sub.UnsubscribedAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &sub, nil
}

// countSubscriptionsFor counts subscriptions for a given email and campaign
func countSubscriptionsFor(db *sql.DB, email, campaignID string) int {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM email_subscriptions 
		WHERE email = ? AND campaign_id = ?
	`, email, campaignID).Scan(&count)

	if err != nil {
		return 0
	}

	return count
}

// countUserSubscriptionsFor counts all subscriptions for a user
func countUserSubscriptionsFor(db *sql.DB, userID int64) int {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM email_subscriptions 
		WHERE user_id = ?
	`, userID).Scan(&count)

	if err != nil {
		return 0
	}

	return count
}

// deleteSubscriptionFor deletes a subscription for cleanup
func deleteSubscriptionFor(db *sql.DB, email string) error {
	_, err := db.Exec(`
		DELETE FROM email_subscriptions 
		WHERE email = ?
	`, email)

	return err
}

// printSubscriptionInfo prints the details of a subscription
func printSubscriptionInfo(t *testing.T, sub *EmailSubscriptionInfo) {
	t.Log("Subscription details:")
	t.Log("--------------------")
	t.Logf("ID:             %d", sub.ID)
	t.Logf("Email:          %s", sub.Email)
	t.Logf("Campaign:       %s", sub.CampaignID)
	t.Logf("UserID:         %v", sub.UserID)
	t.Logf("Status:         %s", sub.Status)
	t.Logf("LastEmailSent:  %d", sub.LastEmailSent)
	t.Logf("SubscribedAt:   %v", sub.SubscribedAt)
	t.Logf("UnsubscribedAt: %v", sub.UnsubscribedAt)
	t.Logf("CreatedAt:      %v", sub.CreatedAt)
	t.Logf("UpdatedAt:      %v", sub.UpdatedAt)
	t.Log("--------------------")
}
