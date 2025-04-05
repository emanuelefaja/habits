package email

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"time"
)

// EmailCampaign represents an email campaign with its metadata and email sequence
type EmailCampaign struct {
	ID            string
	Name          string
	Description   string
	Emoji         string
	AutoSubscribe bool // Whether new users should be auto-subscribed
	Emails        []CampaignEmail
}

// CampaignEmail represents a single email in a campaign sequence
type CampaignEmail struct {
	Number       int
	Subject      string
	Title        string
	TemplateName string // e.g., "courses/digital-detox/1-welcome"
	SendDay      int    // Days after subscription (0 = immediate)
}

// EmailSubscription represents a user's subscription to a campaign
type EmailSubscription struct {
	ID             int
	UserID         sql.NullInt64 // NULL for non-registered users
	Email          string
	CampaignID     string
	SubscribedAt   time.Time
	Status         string // "active" or "unsubscribed"
	LastEmailSent  int    // The last email number sent
	UnsubscribedAt sql.NullTime
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// EmailSend represents a record of an email sent to a subscriber
type EmailSend struct {
	ID             int
	SubscriptionID int
	EmailNumber    int
	TemplateName   string
	Subject        string
	Status         string // "success", "failed", "retry"
	SentAt         time.Time
	ErrorMessage   sql.NullString
	RetryCount     int
	CreatedAt      time.Time
}

// Define all available campaigns
var Campaigns = map[string]EmailCampaign{
	"onboarding": {
		ID:            "onboarding",
		Name:          "Getting Started with Habits",
		Description:   "Learn the basics of using Habits to build consistent routines.",
		Emoji:         "🚀",
		AutoSubscribe: true, // Auto-subscribe all new users
		Emails: []CampaignEmail{
			{
				Number:       1,
				Subject:      "Welcome to Habits!",
				Title:        "Welcome to Your Habit Journey",
				TemplateName: "courses/onboarding/1-welcome",
				SendDay:      0, // Send immediately after registration
			},
			{
				Number:       2,
				Subject:      "Track Your First Week of Habits",
				Title:        "Building Momentum: Your First Week",
				TemplateName: "courses/onboarding/2-first-week",
				SendDay:      1, // Send 1 day after registration
			},
			{
				Number:       3,
				Subject:      "Setting Effective Goals",
				Title:        "The Power of Goal Setting",
				TemplateName: "courses/onboarding/3-goals",
				SendDay:      3, // Send 3 days after registration
			},
			{
				Number:       4,
				Subject:      "Creating a Habit Routine",
				Title:        "Building a Routine for Success",
				TemplateName: "courses/onboarding/4-routine",
				SendDay:      7, // Send 7 days after registration
			},
		},
	},
	"digital-detox": {
		ID:            "digital-detox",
		Name:          "Digital Detox",
		Description:   "Break free from digital dependence and reclaim your focus.",
		Emoji:         "📱",
		AutoSubscribe: false, // Opt-in only
		Emails: []CampaignEmail{
			{
				Number:       1,
				Subject:      "Day 1: The Phone Addiction Pandemic",
				Title:        "Day 1: The Phone Addiction Pandemic",
				TemplateName: "courses/digital-detox/1-phone-addiction",
				SendDay:      0, // Send immediately
			},
		},
	},
	"phone-addiction": {
		ID:            "phone-addiction",
		Name:          "Breaking Phone Addiction",
		Description:   "Break the cycle of smartphone addiction and regain control of your time and attention.",
		Emoji:         "📵",
		AutoSubscribe: false, // Opt-in only
		Emails: []CampaignEmail{
			{
				Number:       1,
				Subject:      "Day 1: The Phone Addiction Pandemic",
				Title:        "Day 1: The Phone Addiction Pandemic",
				TemplateName: "courses/phone-addiction/1-phone-addiction",
				SendDay:      0, // Send immediately
			},
			{
				Number:       2,
				Subject:      "Day 2: How Social Media Exploits Your Caveman Brain",
				Title:        "Day 2: How Social Media Exploits Your Caveman Brain",
				TemplateName: "courses/phone-addiction/2-caveman-brain",
				SendDay:      1, // Send 1 day after signup
			},
			{
				Number:       3,
				Subject:      "Day 3: 3 Ways to Break Free from Phone Addiction",
				Title:        "Day 3: 3 Ways to Break Free from Phone Addiction",
				TemplateName: "courses/phone-addiction/3-break-free-phone-addiction",
				SendDay:      2, // Send 2 days after signup
			},
			{
				Number:       4,
				Subject:      "Day 4: Remix Your Routine",
				Title:        "Day 4: Remix Your Routine",
				TemplateName: "courses/phone-addiction/4-routine",
				SendDay:      3, // Send 3 days after signup
			},
			{
				Number:       5,
				Subject:      "Day 5: Do a 24-Hour Digital Detox",
				Title:        "Day 5: Do a 24-Hour Digital Detox",
				TemplateName: "courses/phone-addiction/5-detox",
				SendDay:      4, // Send 4 days after signup
			},
		},
	},
}

// CampaignManager handles database operations for email campaigns
type CampaignManager struct {
	db       *sql.DB
	emailSvc EmailService
}

// NewCampaignManager creates a new campaign manager
func NewCampaignManager(db *sql.DB, emailSvc EmailService) *CampaignManager {
	return &CampaignManager{
		db:       db,
		emailSvc: emailSvc,
	}
}

// GetCampaign returns a campaign by ID or an error if not found
func GetCampaign(campaignID string) (EmailCampaign, error) {
	campaign, exists := Campaigns[campaignID]
	if !exists {
		return EmailCampaign{}, fmt.Errorf("campaign with ID %s not found", campaignID)
	}
	return campaign, nil
}

// GetAllCampaigns returns a slice of all available campaigns
func GetAllCampaigns() []EmailCampaign {
	campaigns := make([]EmailCampaign, 0, len(Campaigns))
	for _, campaign := range Campaigns {
		campaigns = append(campaigns, campaign)
	}
	return campaigns
}

// GetAutoSubscribeCampaigns returns all campaigns that should auto-subscribe new users
func GetAutoSubscribeCampaigns() []EmailCampaign {
	var autoSubscribeCampaigns []EmailCampaign
	for _, campaign := range Campaigns {
		if campaign.AutoSubscribe {
			autoSubscribeCampaigns = append(autoSubscribeCampaigns, campaign)
		}
	}
	return autoSubscribeCampaigns
}

// GenerateUnsubscribeLink creates a unique unsubscribe link for a campaign subscription
func GenerateUnsubscribeLink(email string, campaignID string, token string) string {
	// Check for explicit BASE_URL first
	baseURL := os.Getenv("BASE_URL")

	// If no BASE_URL defined, use environment-specific default
	if baseURL == "" {
		// Use production URL if APP_ENV is production
		if os.Getenv("APP_ENV") == "production" {
			baseURL = "https://habits.co"
		} else {
			// Default to localhost for development/testing
			baseURL = "http://localhost:8080"
		}
	}

	// Use url.QueryEscape to properly encode the parameters
	emailEncoded := url.QueryEscape(email)
	campaignEncoded := url.QueryEscape(campaignID)
	tokenEncoded := url.QueryEscape(token)

	return fmt.Sprintf("%s/unsubscribe?email=%s&campaign=%s&token=%s",
		baseURL, emailEncoded, campaignEncoded, tokenEncoded)
}

// CampaignEmailData returns the data needed for a campaign email template
func CampaignEmailData(firstName, email, campaignID string, emailNumber int) (map[string]interface{}, error) {
	campaign, err := GetCampaign(campaignID)
	if err != nil {
		return nil, err
	}

	// Find the specific email in campaign
	var campaignEmail CampaignEmail
	found := false
	for _, e := range campaign.Emails {
		if e.Number == emailNumber {
			campaignEmail = e
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("email number %d not found in campaign %s", emailNumber, campaignID)
	}

	// Use the secure token from the database
	unsubscribeToken := ""

	// Get the database path from environment
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "habits.db" // Default fallback
		log.Printf("Warning: DATABASE_PATH not set, using default path 'habits.db'")
	}

	log.Printf("Opening database at: %s", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database to retrieve token: %v", err)
		// Don't fall back to a timestamp - log the error and return it
		return nil, fmt.Errorf("could not open database to get unsubscribe token: %w", err)
	}
	defer db.Close()

	// First check if the subscription exists and what the token is
	log.Printf("Querying token for email=%s, campaign=%s", email, campaignID)

	err = db.QueryRow(`
		SELECT token 
		FROM email_subscriptions
		WHERE email = ?
		AND campaign_id = ?
		AND status = 'active'
	`, email, campaignID).Scan(&unsubscribeToken)

	if err != nil {
		log.Printf("Error retrieving token from database: %v", err)

		// Check if subscription exists at all
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM email_subscriptions 
				WHERE email = ? AND campaign_id = ?
			)
		`, email, campaignID).Scan(&exists)

		if err != nil {
			log.Printf("Error checking if subscription exists: %v", err)
			return nil, fmt.Errorf("database error: %w", err)
		}

		if !exists {
			log.Printf("No subscription found for email=%s, campaign=%s - creating one", email, campaignID)

			// Create a real subscription with a proper secure token
			// Use the same secure token generation as in real subscriptions
			unsubscribeToken = generateSecureToken()
			log.Printf("Generated secure token for new subscription: %s", unsubscribeToken)

			// Insert the subscription with this token
			_, err = db.Exec(`
				INSERT INTO email_subscriptions (
					user_id, email, campaign_id, token, subscribed_at, 
					status, last_email_sent, created_at, updated_at
				) VALUES (NULL, ?, ?, ?, CURRENT_TIMESTAMP, 'active', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, email, campaignID, unsubscribeToken)

			if err != nil {
				log.Printf("Error creating subscription: %v", err)
				return nil, fmt.Errorf("could not create subscription: %w", err)
			}

			log.Printf("Successfully created subscription with secure token")
		} else {
			// Subscription exists but is not active or couldn't get token
			log.Printf("Subscription exists but couldn't get token - creating a new one")

			// Generate a new secure token and update the subscription
			unsubscribeToken = generateSecureToken()

			_, err = db.Exec(`
				UPDATE email_subscriptions
				SET token = ?,
				    status = 'active',
				    updated_at = CURRENT_TIMESTAMP
				WHERE email = ? AND campaign_id = ?
			`, unsubscribeToken, email, campaignID)

			if err != nil {
				log.Printf("Error updating subscription token: %v", err)
				return nil, fmt.Errorf("could not update subscription token: %w", err)
			}

			log.Printf("Successfully updated subscription with new secure token: %s", unsubscribeToken)
		}
	} else {
		log.Printf("Successfully retrieved token from database: %s", unsubscribeToken)
	}

	// Create the data for the email template
	data := map[string]interface{}{
		"FirstName":       firstName,
		"Email":           email,
		"Subject":         campaignEmail.Subject,
		"Title":           campaignEmail.Title,
		"AppName":         "The Habits Company",
		"CampaignName":    campaign.Name,
		"CampaignEmoji":   campaign.Emoji,
		"CampaignID":      campaign.ID,
		"UnsubscribeLink": GenerateUnsubscribeLink(email, campaignID, unsubscribeToken),
	}

	log.Printf("📧 Prepared campaign email data for %s, email #%d with token %s", campaignID, emailNumber, unsubscribeToken)
	return data, nil
}

// SubscribeUser subscribes a user to a campaign
func (cm *CampaignManager) SubscribeUser(email string, campaignID string, userID int) error {
	// Check if campaign exists
	if _, err := GetCampaign(campaignID); err != nil {
		return err
	}

	// First check if user is already subscribed
	var exists bool
	var status string
	var id int

	query := `
	SELECT EXISTS(
		SELECT 1 FROM email_subscriptions 
		WHERE email = ? AND campaign_id = ?
	), status, id 
	FROM email_subscriptions
	WHERE email = ? AND campaign_id = ?
	LIMIT 1`

	err := cm.db.QueryRow(query, email, campaignID, email, campaignID).Scan(&exists, &status, &id)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error checking existing subscription: %w", err)
	}

	// If already subscribed and active, do nothing
	if exists && status == "active" {
		log.Printf("User %s already subscribed to campaign %s", email, campaignID)
		return nil
	}

	// If exists but unsubscribed, reactivate
	if exists && status == "unsubscribed" {
		log.Printf("Reactivating subscription for %s to campaign %s", email, campaignID)

		updateQuery := `
		UPDATE email_subscriptions 
		SET status = 'active', 
		    last_email_sent = 0, 
		    unsubscribed_at = NULL,
		    subscribed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

		_, err = cm.db.Exec(updateQuery, id)
		return err
	}

	// Otherwise create new subscription
	var userIDValue sql.NullInt64
	if userID > 0 {
		userIDValue.Int64 = int64(userID)
		userIDValue.Valid = true
	}

	token := generateSecureToken()

	insertQuery := `
	INSERT INTO email_subscriptions (
		user_id, email, campaign_id, token, subscribed_at, 
		status, last_email_sent, created_at, updated_at
	) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, 'active', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err = cm.db.Exec(insertQuery, userIDValue, email, campaignID, token)
	if err != nil {
		return fmt.Errorf("error creating subscription: %w", err)
	}

	log.Printf("Successfully subscribed %s to campaign %s", email, campaignID)
	return nil
}

// UnsubscribeUser unsubscribes a user from a campaign
func (cm *CampaignManager) UnsubscribeUser(email string, campaignID string) error {
	query := `
	UPDATE email_subscriptions 
	SET status = 'unsubscribed', 
	    unsubscribed_at = CURRENT_TIMESTAMP,
	    updated_at = CURRENT_TIMESTAMP
	WHERE email = ? AND campaign_id = ? AND status = 'active'`

	result, err := cm.db.Exec(query, email, campaignID)
	if err != nil {
		return fmt.Errorf("error unsubscribing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no active subscription found for %s in campaign %s", email, campaignID)
	}

	log.Printf("Successfully unsubscribed %s from campaign %s", email, campaignID)
	return nil
}

// AutoSubscribeUser subscribes a user to all auto-subscribe campaigns
func (cm *CampaignManager) AutoSubscribeUser(email string, userID int) error {
	autoSubscribeCampaigns := GetAutoSubscribeCampaigns()

	for _, campaign := range autoSubscribeCampaigns {
		if err := cm.SubscribeUser(email, campaign.ID, userID); err != nil {
			log.Printf("Error auto-subscribing user %s to campaign %s: %v", email, campaign.ID, err)
			// Continue with other campaigns even if one fails
			continue
		}
	}

	return nil
}

// GetPendingEmails returns subscriptions that need emails sent
func (cm *CampaignManager) GetPendingEmails() ([]EmailSubscription, error) {
	query := `
	SELECT id, user_id, email, campaign_id, subscribed_at, status, last_email_sent, 
	       unsubscribed_at, created_at, updated_at
	FROM email_subscriptions 
	WHERE status = 'active'`

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying pending emails: %w", err)
	}
	defer rows.Close()

	var subscriptions []EmailSubscription
	for rows.Next() {
		var sub EmailSubscription
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Email, &sub.CampaignID, &sub.SubscribedAt,
			&sub.Status, &sub.LastEmailSent, &sub.UnsubscribedAt, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning subscription: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// LogEmailSend records an email send in the database
func (cm *CampaignManager) LogEmailSend(subscriptionID int, emailNumber int, templateName, subject, status string, errorMsg string) error {
	var errorValue sql.NullString
	if errorMsg != "" {
		errorValue.String = errorMsg
		errorValue.Valid = true
	}

	query := `
	INSERT INTO email_sends (
		subscription_id, email_number, template_name, subject, status, sent_at, 
		error_message, retry_count, created_at
	) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, 0, CURRENT_TIMESTAMP)`

	_, err := cm.db.Exec(query, subscriptionID, emailNumber, templateName, subject, status, errorValue)
	if err != nil {
		return fmt.Errorf("error logging email send: %w", err)
	}

	// If successful, update last_email_sent in subscription
	if status == "success" {
		updateQuery := `
		UPDATE email_subscriptions 
		SET last_email_sent = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

		_, err = cm.db.Exec(updateQuery, emailNumber, subscriptionID)
		if err != nil {
			return fmt.Errorf("error updating subscription: %w", err)
		}
	}

	return nil
}

// SendCampaignEmail sends a specific email to a subscriber
func (cm *CampaignManager) SendCampaignEmail(subscription EmailSubscription, emailNumber int) error {
	campaign, err := GetCampaign(subscription.CampaignID)
	if err != nil {
		return err
	}

	// Find the specific email in campaign
	var campaignEmail CampaignEmail
	found := false
	for _, e := range campaign.Emails {
		if e.Number == emailNumber {
			campaignEmail = e
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("email number %d not found in campaign %s", emailNumber, subscription.CampaignID)
	}

	// Get first name from users table if this is a registered user
	firstName := "there" // Default
	if subscription.UserID.Valid {
		var fname string
		err := cm.db.QueryRow("SELECT first_name FROM users WHERE id = ?", subscription.UserID.Int64).Scan(&fname)
		if err == nil && fname != "" {
			firstName = fname
		}
	}

	// Prepare email data
	emailData, err := CampaignEmailData(firstName, subscription.Email, subscription.CampaignID, emailNumber)
	if err != nil {
		return err
	}

	// Create email template
	template := EmailTemplate{
		Name:    campaignEmail.TemplateName,
		Subject: campaignEmail.Subject,
	}

	// Try to send the email
	err = cm.emailSvc.SendTypedEmail(subscription.Email, template, emailData)
	status := "success"
	errorMsg := ""

	if err != nil {
		status = "failed"
		errorMsg = err.Error()
		log.Printf("❌ Failed to send campaign email: %v", err)
	} else {
		log.Printf("✅ Successfully sent campaign email #%d to %s", emailNumber, subscription.Email)
	}

	// Log the send attempt
	return cm.LogEmailSend(
		subscription.ID,
		emailNumber,
		campaignEmail.TemplateName,
		campaignEmail.Subject,
		status,
		errorMsg,
	)
}

// SendPendingCampaignEmails checks for and sends any pending campaign emails
func (cm *CampaignManager) SendPendingCampaignEmails() error {
	// Use default batch size of 100
	return cm.SendPendingCampaignEmailsWithLimit(100)
}

// SendPendingCampaignEmailsWithLimit checks for and sends any pending campaign emails with a specified limit
func (cm *CampaignManager) SendPendingCampaignEmailsWithLimit(batchSize int) error {
	subscriptions, err := cm.GetPendingEmails()
	if err != nil {
		return err
	}

	log.Printf("Found %d active subscriptions to process", len(subscriptions))

	// Track how many emails we've sent in this batch
	emailsSent := 0
	emailLimit := batchSize // Use the provided batch size

	for _, sub := range subscriptions {
		// Stop if we've hit our batch limit
		if emailsSent >= emailLimit {
			log.Printf("Reached email batch limit of %d", emailLimit)
			break
		}

		campaign, err := GetCampaign(sub.CampaignID)
		if err != nil {
			log.Printf("Error getting campaign %s: %v", sub.CampaignID, err)
			continue
		}

		// Calculate which emails should have been sent by now
		daysSinceSubscription := int(time.Since(sub.SubscribedAt).Hours() / 24)

		// Find the next email to send
		for _, email := range campaign.Emails {
			// Skip emails we've already sent
			if email.Number <= sub.LastEmailSent {
				continue
			}

			// Skip emails that aren't due yet
			if email.SendDay > daysSinceSubscription {
				continue
			}

			// We found an email that needs to be sent
			err := cm.SendCampaignEmail(sub, email.Number)
			if err != nil {
				log.Printf("Error sending email #%d for campaign %s to %s: %v",
					email.Number, sub.CampaignID, sub.Email, err)
				continue
			}

			emailsSent++

			// Only send one email per subscription per run
			break
		}
	}

	log.Printf("Completed sending batch of %d campaign emails", emailsSent)
	return nil
}

// GetUserSubscriptions returns all subscriptions for a user
func (cm *CampaignManager) GetUserSubscriptions(userID int) ([]EmailSubscription, error) {
	query := `
	SELECT id, user_id, email, campaign_id, subscribed_at, status, last_email_sent, 
	       unsubscribed_at, created_at, updated_at
	FROM email_subscriptions 
	WHERE user_id = ? AND status = 'active'`

	rows, err := cm.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying user subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []EmailSubscription
	for rows.Next() {
		var sub EmailSubscription
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Email, &sub.CampaignID, &sub.SubscribedAt,
			&sub.Status, &sub.LastEmailSent, &sub.UnsubscribedAt, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning subscription: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// UpdateSubscriptionStatus updates the status of a user's subscription
func (cm *CampaignManager) UpdateSubscriptionStatus(userID int, campaignID, status string) error {
	// Validate status
	if status != "active" && status != "paused" {
		return fmt.Errorf("invalid status: %s", status)
	}

	query := `
	UPDATE email_subscriptions 
	SET status = ?, 
	    updated_at = CURRENT_TIMESTAMP
	WHERE user_id = ? AND campaign_id = ?`

	result, err := cm.db.Exec(query, status, userID, campaignID)
	if err != nil {
		return fmt.Errorf("error updating subscription status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no subscription found for user %d and campaign %s", userID, campaignID)
	}

	log.Printf("Successfully updated subscription status to %s for user %d and campaign %s", status, userID, campaignID)
	return nil
}

func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// HandleUserDeletion cleans up email subscriptions when a user is deleted
// This preserves email logs while removing the user's personal information
func (cm *CampaignManager) HandleUserDeletion(userID int64) error {
	// First, get the user's email from subscriptions for logging purposes
	var userEmail string
	err := cm.db.QueryRow(`
		SELECT email FROM email_subscriptions 
		WHERE user_id = ? 
		LIMIT 1`, userID).Scan(&userEmail)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error getting user email: %w", err)
	}

	// Start a transaction
	tx, err := cm.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Update email_subscriptions to anonymize the email
	// We'll set status to 'unsubscribed' and use a placeholder for the email
	anonymizedEmail := fmt.Sprintf("deleted-user-%d@anonymous.com", userID)

	_, err = tx.Exec(`
		UPDATE email_subscriptions 
		SET status = 'unsubscribed',
		    email = ?,
		    unsubscribed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?`,
		anonymizedEmail, userID)

	if err != nil {
		return fmt.Errorf("error anonymizing email subscriptions: %w", err)
	}

	// Note: We don't need to modify email_sends table as it references subscription_id
	// and doesn't contain PII directly

	log.Printf("Successfully cleaned up email subscriptions for deleted user ID %d", userID)

	// Commit the transaction
	return tx.Commit()
}

// ValidateUnsubscribeToken checks if the provided token is valid for the given email and campaign
func (cm *CampaignManager) ValidateUnsubscribeToken(email, campaignID, token string) (bool, error) {
	var storedToken string
	err := cm.db.QueryRow(`
		SELECT token 
		FROM email_subscriptions
		WHERE email = ? 
		AND campaign_id = ?
		AND status = 'active'
	`, email, campaignID).Scan(&storedToken)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No active subscription found for email=%s, campaign=%s", email, campaignID)
			return false, fmt.Errorf("no active subscription found")
		}
		log.Printf("Database error while validating token: %v", err)
		return false, fmt.Errorf("database error: %v", err)
	}

	valid := token == storedToken
	if !valid {
		log.Printf("Token mismatch: provided=%s vs stored=%s", token, storedToken)
	} else {
		log.Printf("Token validated successfully")
	}
	return valid, nil
}
