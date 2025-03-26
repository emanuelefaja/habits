package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"mad/models"
	"mad/models/email"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("📧 Email Template Tester")
	fmt.Println("========================")

	// Find the root directory and load .env file
	rootDir, err := findRootDirForEmailTest()
	if err != nil {
		fmt.Printf("Error finding root directory: %v\n", err)
		return
	}

	// Load .env file from root directory
	if err := godotenv.Load(filepath.Join(rootDir, ".env")); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		return
	}

	// Override APP_ENV to force email sending
	originalAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "production")
	fmt.Println("\n⚠️  WARNING: Temporarily setting APP_ENV to 'production' to allow actual email sending")
	fmt.Println("    Emails will be ACTUALLY SENT to the recipient address")
	defer func() {
		// Restore original APP_ENV when the program exits
		os.Setenv("APP_ENV", originalAppEnv)
	}()

	// Get user input
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\nEnter recipient email: ")
	to, _ := reader.ReadString('\n')
	to = strings.TrimSpace(to)

	// Template selection
	fmt.Println("\nSelect email template to test:")
	fmt.Println("1. Welcome Email")
	fmt.Println("2. Password Reset Email")
	fmt.Println("3. Password Reset Success Email")
	fmt.Println("4. Daily Habit Reminder Email")
	fmt.Println("5. First Habit Email")
	fmt.Println("6. Campaign Emails")
	fmt.Print("\nEnter selection (1-6): ")
	templateChoice, _ := reader.ReadString('\n')
	templateChoice = strings.TrimSpace(templateChoice)

	// Get SMTP settings from environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	fromName := os.Getenv("SMTP_FROM_NAME")

	// Create email service
	emailService, err := email.NewSMTPEmailService(email.SMTPConfig{
		Host:        smtpHost,
		Port:        587,
		Username:    smtpUsername,
		Password:    smtpPassword,
		FromName:    fromName,
		FromEmail:   fromEmail,
		TemplateDir: filepath.Join(rootDir, "ui/email"),
		Secure:      true,
		RequireTLS:  true,
	})
	if err != nil {
		fmt.Printf("Error creating email service: %v\n", err)
		return
	}

	// Connect to database for user and habit data
	db, err := sql.Open("sqlite3", filepath.Join(rootDir, "habits.db"))
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Create and set up campaign manager for unsubscribe testing
	campaignManager := email.NewCampaignManager(db, emailService)
	if smtpService, ok := emailService.(*email.SMTPEmailService); ok {
		smtpService.SetCampaignManager(campaignManager)
	} else {
		fmt.Println("⚠️  Warning: Could not set campaign manager - unsubscribe testing may not work properly")
	}

	// Process based on template choice
	switch templateChoice {
	case "1":
		// Welcome Email
		fmt.Print("Enter first name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)

		err = emailService.SendWelcomeEmail(to, firstName)
		if err != nil {
			fmt.Printf("Error sending welcome email: %v\n", err)
			return
		}
		fmt.Println("✅ Welcome email sent successfully!")

	case "2":
		// Password Reset Email
		fmt.Print("Enter first name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)

		resetLink := "https://habits.co/reset?token=test-token-12345"
		expiry := time.Now().Add(24 * time.Hour) // Set expiry to 24 hours from now
		err = emailService.SendPasswordResetEmail(to, resetLink, expiry)
		if err != nil {
			fmt.Printf("Error sending password reset email: %v\n", err)
			return
		}
		fmt.Println("✅ Password reset email sent successfully!")

	case "3":
		// Password Reset Success Email
		fmt.Print("Enter first name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)

		err = emailService.SendPasswordResetSuccessEmail(to, firstName)
		if err != nil {
			fmt.Printf("Error sending password reset success email: %v\n", err)
			return
		}
		fmt.Println("✅ Password reset success email sent successfully!")

	case "4":
		// Daily Habit Reminder Email
		// Try to find user by email
		var userID int64
		err := db.QueryRow("SELECT id FROM users WHERE email = ?", to).Scan(&userID)
		if err != nil {
			fmt.Println("User not found with that email. Using test data instead.")
			// Use test data
			fmt.Print("Enter first name: ")
			firstName, _ := reader.ReadString('\n')
			firstName = strings.TrimSpace(firstName)

			// Create test habits
			habits := []email.HabitInfo{
				{Name: "Drink Water", Emoji: "💧"},
				{Name: "Exercise", Emoji: "🏃"},
				{Name: "Read", Emoji: "📚"},
			}

			// Get a random quote
			quote, err := models.GetRandomQuoteForEmail()
			if err != nil {
				fmt.Printf("Warning: Error getting random quote: %v\n", err)
				// Use a default quote if there's an error
				quote = email.QuoteInfo{
					Text:   "We are what we repeatedly do. Excellence, then, is not an act, but a habit.",
					Author: "Aristotle",
				}
			}

			err = emailService.SendReminderEmail(to, firstName, habits, quote)
			if err != nil {
				fmt.Printf("Error sending reminder email: %v\n", err)
				return
			}
		} else {
			// Get user data
			user, err := models.GetUserByID(db, userID)
			if err != nil {
				fmt.Printf("Error getting user: %v\n", err)
				return
			}

			// Get user's habits
			habits, err := models.GetHabitsByUserID(db, int(userID))
			if err != nil {
				fmt.Printf("Error getting habits: %v\n", err)
				return
			}

			// Convert habits to HabitInfo format
			habitInfos := make([]email.HabitInfo, 0, len(habits))
			for _, habit := range habits {
				habitInfos = append(habitInfos, email.HabitInfo{
					Name:  habit.Name,
					Emoji: habit.Emoji,
				})
			}

			// Get a random quote
			quote, err := models.GetRandomQuoteForEmail()
			if err != nil {
				fmt.Printf("Warning: Error getting random quote: %v\n", err)
				// Use a default quote if there's an error
				quote = email.QuoteInfo{
					Text:   "We are what we repeatedly do. Excellence, then, is not an act, but a habit.",
					Author: "Aristotle",
				}
			}

			// Send the email
			err = emailService.SendReminderEmail(to, user.FirstName, habitInfos, quote)
			if err != nil {
				fmt.Printf("Error sending reminder email: %v\n", err)
				return
			}
		}
		fmt.Println("✅ Reminder email sent successfully!")

	case "5":
		// First Habit Email
		fmt.Print("Enter first name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)

		// Get a random quote
		quote, err := models.GetRandomQuoteForEmail()
		if err != nil {
			fmt.Printf("Warning: Error getting random quote: %v\n", err)
			// Use a default quote if there's an error
			quote = email.QuoteInfo{
				Text:   "The secret of getting ahead is getting started.",
				Author: "Mark Twain",
			}
		}

		err = emailService.SendFirstHabitEmail(to, firstName, quote)
		if err != nil {
			fmt.Printf("Error sending first habit email: %v\n", err)
			return
		}
		fmt.Println("✅ First habit email sent successfully!")

	case "6":
		// Campaign Emails
		// Get all available campaigns
		campaigns := email.GetAllCampaigns()

		// Display available campaigns
		fmt.Println("\nAvailable campaigns:")
		for i, campaign := range campaigns {
			fmt.Printf("%d. %s %s - %s\n", i+1, campaign.Emoji, campaign.Name, campaign.Description)
		}

		// Ask user to select a campaign
		fmt.Print("\nSelect a campaign (1-" + strconv.Itoa(len(campaigns)) + "): ")
		campaignChoice, _ := reader.ReadString('\n')
		campaignChoice = strings.TrimSpace(campaignChoice)

		campaignIndex, err := strconv.Atoi(campaignChoice)
		if err != nil || campaignIndex < 1 || campaignIndex > len(campaigns) {
			fmt.Printf("Invalid campaign selection: %s\n", campaignChoice)
			return
		}

		// Get the selected campaign
		selectedCampaign := campaigns[campaignIndex-1]

		// Ask if user wants to create a test subscription for unsubscribe testing
		fmt.Print("\nDo you want to create a real subscription for unsubscribe testing? (y/n): ")
		createSubscription, _ := reader.ReadString('\n')
		createSubscription = strings.TrimSpace(createSubscription)

		// Create a real subscription for testing the unsubscribe flow if requested
		if strings.ToLower(createSubscription) == "y" {
			fmt.Println("\nCreating test subscription for email:", to)

			// Delete any existing subscription first to start fresh
			_, err = db.Exec("DELETE FROM email_subscriptions WHERE email = ? AND campaign_id = ?",
				to, selectedCampaign.ID)
			if err != nil {
				fmt.Printf("Warning: Failed to clean up existing subscription: %v\n", err)
			}

			// Create a new subscription
			err = campaignManager.SubscribeUser(to, selectedCampaign.ID, 0) // 0 for non-user subscription
			if err != nil {
				fmt.Printf("Warning: Failed to create test subscription: %v\n", err)
			} else {
				fmt.Println("✅ Created test subscription for unsubscribe testing")
				fmt.Println("   You can click the unsubscribe link in the email to test the unsubscribe flow")

				// Get the subscription details to show the token
				var token string
				err = db.QueryRow(
					"SELECT token FROM email_subscriptions WHERE email = ? AND campaign_id = ?",
					to, selectedCampaign.ID).Scan(&token)
				if err == nil {
					fmt.Println("   Unsubscribe token:", token)
				}
			}
		}

		// Display available emails in the campaign
		fmt.Printf("\nEmails in %s campaign:\n", selectedCampaign.Name)
		for _, campaignEmail := range selectedCampaign.Emails {
			fmt.Printf("%d. %s (Day %d)\n", campaignEmail.Number, campaignEmail.Subject, campaignEmail.SendDay)
		}

		// Ask user to select an email
		fmt.Print("\nSelect an email number: ")
		emailChoice, _ := reader.ReadString('\n')
		emailChoice = strings.TrimSpace(emailChoice)

		emailNumber, err := strconv.Atoi(emailChoice)
		if err != nil {
			fmt.Printf("Invalid email selection: %s\n", emailChoice)
			return
		}

		// Validate the email number exists in the campaign
		var selectedEmail email.CampaignEmail
		found := false
		for _, e := range selectedCampaign.Emails {
			if e.Number == emailNumber {
				selectedEmail = e
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Email number %d not found in campaign %s\n", emailNumber, selectedCampaign.Name)
			return
		}

		// Ask for first name to use in the email
		fmt.Print("\nEnter first name to use in email: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)
		if firstName == "" {
			firstName = "Manny" // Default if no name provided
		}

		// Prepare email data
		data, err := email.CampaignEmailData(firstName, to, selectedCampaign.ID, emailNumber)
		if err != nil {
			fmt.Printf("Error preparing campaign email data: %v\n", err)
			return
		}

		// Send the campaign email
		template := email.EmailTemplate{
			Name:    selectedEmail.TemplateName,
			Subject: selectedEmail.Subject,
		}

		err = emailService.SendTypedEmail(to, template, data)
		if err != nil {
			fmt.Printf("Error sending campaign email: %v\n", err)
			return
		}

		fmt.Printf("✅ Campaign email '%s' from '%s' sent successfully!\n", selectedEmail.Subject, selectedCampaign.Name)
		fmt.Println("   Check your email to view it and test the unsubscribe link if you created a subscription")

	default:
		fmt.Printf("Invalid template choice: %s\n", templateChoice)
		return
	}
}

// findRootDirForEmailTest attempts to find the root directory of the project
func findRootDirForEmailTest() (string, error) {
	// Try to find the root directory by looking for common files
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// First check if we're already in the root directory
	if _, err := os.Stat(filepath.Join(currentDir, ".env")); err == nil {
		return currentDir, nil
	}

	// If not, try going up one level (assuming we're in cmd/emailtest)
	rootDir := filepath.Join(currentDir, "../..")
	if _, err := os.Stat(filepath.Join(rootDir, ".env")); err == nil {
		return rootDir, nil
	}

	// If still not found, return an error
	return "", fmt.Errorf("could not find root directory with .env file")
}
