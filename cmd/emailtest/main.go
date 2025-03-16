package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"mad/models"
	"mad/models/email"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("📧 Email Template Tester")
	fmt.Println("========================")

	// Find the root directory and load .env file
	rootDir, err := findRootDir()
	if err != nil {
		fmt.Printf("Error finding root directory: %v\n", err)
		return
	}

	// Load .env file from root directory
	if err := godotenv.Load(filepath.Join(rootDir, ".env")); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		return
	}

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
	fmt.Print("\nEnter selection (1-5): ")
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

	default:
		fmt.Printf("Invalid template choice: %s\n", templateChoice)
		return
	}
}

// findRootDir attempts to find the root directory of the project
func findRootDir() (string, error) {
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
