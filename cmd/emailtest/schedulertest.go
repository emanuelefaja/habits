package main

import (
	"database/sql"
	"flag"
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

// TestEmailService wraps an email service and redirects all emails to a test recipient
type TestEmailService struct {
	baseService   email.EmailService
	testRecipient string
	emailsSent    map[string]int // Tracks email types sent
}

// Create a new test email service
func newTestEmailService(baseService email.EmailService, testRecipient string) *TestEmailService {
	return &TestEmailService{
		baseService:   baseService,
		testRecipient: testRecipient,
		emailsSent:    make(map[string]int),
	}
}

// SendWelcomeEmail redirects the welcome email to the test recipient
func (s *TestEmailService) SendWelcomeEmail(to, firstName string) error {
	fmt.Printf("📧 Sending welcome email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["welcome"]++
	return s.baseService.SendWelcomeEmail(s.testRecipient, firstName)
}

// SendPasswordResetEmail redirects the password reset email to the test recipient
func (s *TestEmailService) SendPasswordResetEmail(to, resetLink string, expiry time.Time) error {
	fmt.Printf("📧 Sending password reset email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["password_reset"]++

	// Add original recipient to subject
	modifiedLink := resetLink + "&original_recipient=" + to
	return s.baseService.SendPasswordResetEmail(s.testRecipient, modifiedLink, expiry)
}

// SendPasswordResetSuccessEmail redirects the password reset success email to the test recipient
func (s *TestEmailService) SendPasswordResetSuccessEmail(to, firstName string) error {
	fmt.Printf("📧 Sending password reset success email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["password_reset_success"]++
	return s.baseService.SendPasswordResetSuccessEmail(s.testRecipient, firstName)
}

// SendReminderEmail redirects the reminder email to the test recipient
func (s *TestEmailService) SendReminderEmail(to, firstName string, habits []email.HabitInfo, quote email.QuoteInfo) error {
	fmt.Printf("📧 Sending daily reminder email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["daily_reminder"]++

	// Create a modified first name that includes the original recipient
	modifiedFirstName := fmt.Sprintf("%s (Original: %s)", firstName, to)
	return s.baseService.SendReminderEmail(s.testRecipient, modifiedFirstName, habits, quote)
}

// SendFirstHabitEmail redirects the first habit email to the test recipient
func (s *TestEmailService) SendFirstHabitEmail(to, firstName string, quote email.QuoteInfo) error {
	fmt.Printf("📧 Sending first habit email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["first_habit"]++

	// Create a modified first name that includes the original recipient
	modifiedFirstName := fmt.Sprintf("%s (Original: %s)", firstName, to)
	return s.baseService.SendFirstHabitEmail(s.testRecipient, modifiedFirstName, quote)
}

// SendSimpleEmail redirects a simple email to the test recipient
func (s *TestEmailService) SendSimpleEmail(to, subject, body string) error {
	fmt.Printf("📧 Sending simple email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["simple"]++

	// Add original recipient to subject
	modifiedSubject := fmt.Sprintf("%s (Original: %s)", subject, to)
	return s.baseService.SendSimpleEmail(s.testRecipient, modifiedSubject, body)
}

// SendTypedEmail redirects a typed email to the test recipient
func (s *TestEmailService) SendTypedEmail(to string, template email.EmailTemplate, data interface{}) error {
	fmt.Printf("📧 Sending typed email to %s (originally for: %s)\n", s.testRecipient, to)
	s.emailsSent["typed"]++

	// If the template has a subject field, try to modify it to include the original recipient
	// This depends on the actual structure of email.EmailTemplate

	return s.baseService.SendTypedEmail(s.testRecipient, template, data)
}

// GetEmailStats returns statistics about sent emails
func (s *TestEmailService) GetEmailStats() map[string]int {
	return s.emailsSent
}

func main() {
	fmt.Println("⏰ Scheduler Simulation Tester")
	fmt.Println("=============================")

	// Parse command line flags
	testEmail := flag.String("email", "", "Email address to send all test emails to")
	days := flag.Int("days", 7, "Number of days to simulate")
	timeScale := flag.Int("timescale", 60, "Time scale factor (seconds per simulated day)")
	flag.Parse()

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

	// Validate and get test email
	recipient := strings.TrimSpace(*testEmail)
	if recipient == "" {
		fmt.Println("Error: Test email is required. Use -email flag.")
		flag.PrintDefaults()
		return
	}

	// Connect to database
	db, err := sql.Open("sqlite3", filepath.Join(rootDir, "habits.db"))
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Create base email service
	baseEmailService, err := createEmailService(rootDir)
	if err != nil {
		fmt.Printf("Error creating email service: %v\n", err)
		return
	}

	// Create test email service wrapper
	testEmailService := newTestEmailService(baseEmailService, recipient)

	// Create scheduler with our test email service
	scheduler := models.NewScheduler(db, testEmailService)

	// Start the simulation
	fmt.Printf("\n🚀 Starting simulation for %d days with emails going to %s\n", *days, recipient)
	fmt.Printf("⏱️  Time scale: %d seconds per day\n\n", *timeScale)

	// Run the simulation
	runSimulation(scheduler, *days, *timeScale)

	// Print summary
	printSummary(testEmailService.GetEmailStats())
}

// runSimulation runs the scheduler simulation for the specified number of days
func runSimulation(scheduler *models.Scheduler, days, timeScale int) {
	// Start the scheduler
	err := scheduler.Start()
	if err != nil {
		fmt.Printf("Error starting scheduler: %v\n", err)
		return
	}
	defer scheduler.Stop()

	// Simulate each day
	for day := 1; day <= days; day++ {
		// Calculate the current simulated date
		simulatedDate := time.Now().AddDate(0, 0, day-1)
		dayOfWeek := simulatedDate.Weekday().String()

		fmt.Printf("\n📅 Day %d (%s): %s\n", day, dayOfWeek, simulatedDate.Format("2006-01-02"))

		// Force run the daily reminder job
		fmt.Println("🔔 Running daily habit reminders...")
		scheduler.RunDailyRemindersNow()

		// On Sundays, also run the weekly job
		if simulatedDate.Weekday() == time.Sunday {
			fmt.Println("🔔 Running weekly first habit reminders (it's Sunday)...")
			scheduler.RunWeeklyFirstHabitRemindersNow()
		}

		// Wait for the specified time scale before moving to the next day
		if day < days {
			fmt.Printf("⏳ Waiting %d seconds before next day...\n", timeScale)
			time.Sleep(time.Duration(timeScale) * time.Second)
		}
	}
}

// printSummary prints a summary of the emails sent during the simulation
func printSummary(stats map[string]int) {
	total := 0
	for _, count := range stats {
		total += count
	}

	fmt.Println("\n📊 Simulation Summary")
	fmt.Println("===================")
	fmt.Printf("Total emails sent: %d\n\n", total)

	// Print counts by email type with emojis
	if count, ok := stats["daily_reminder"]; ok {
		fmt.Printf("📅 Daily reminder emails: %d\n", count)
	}
	if count, ok := stats["first_habit"]; ok {
		fmt.Printf("🌱 First habit emails: %d\n", count)
	}
	if count, ok := stats["welcome"]; ok {
		fmt.Printf("👋 Welcome emails: %d\n", count)
	}
	if count, ok := stats["password_reset"]; ok {
		fmt.Printf("🔑 Password reset emails: %d\n", count)
	}
	if count, ok := stats["password_reset_success"]; ok {
		fmt.Printf("✅ Password reset success emails: %d\n", count)
	}
	if count, ok := stats["simple"]; ok {
		fmt.Printf("📝 Simple emails: %d\n", count)
	}
}

// createEmailService creates the base email service
func createEmailService(rootDir string) (email.EmailService, error) {
	// Get SMTP settings from environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	fromName := os.Getenv("SMTP_FROM_NAME")

	// Create email service
	return email.NewSMTPEmailService(email.SMTPConfig{
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
