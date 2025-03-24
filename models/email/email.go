package email

import "time"

// EmailTemplate represents an email template with its name and subject
type EmailTemplate struct {
	Name    string
	Subject string
}

// Email Data Structures
type WelcomeEmailData struct {
	Username string
	AppName  string
}

type PasswordResetEmailData struct {
	ResetLink   string
	ExpiryHours string
	Username    string
	AppName     string
}

type PasswordResetSuccessEmailData struct {
	Username  string
	AppName   string
	LoginLink string
}

// ReminderEmailData represents data for daily habit reminder emails
type ReminderEmailData struct {
	FirstName string
	Habits    []HabitInfo
	Quote     QuoteInfo
	AppName   string
}

// FirstHabitEmailData represents data for emails to users without habits
type FirstHabitEmailData struct {
	FirstName string
	Quote     QuoteInfo
	AppName   string
}

// HabitInfo represents a habit for display in emails
type HabitInfo struct {
	Name  string
	Emoji string
}

// QuoteInfo represents a motivational quote
type QuoteInfo struct {
	Text   string
	Author string
}

// Predefined Email Templates
var (
	// WelcomeEmail template for new user registration
	WelcomeEmail = EmailTemplate{
		Name:    "welcome",
		Subject: "Welcome to Habits!",
	}

	// PasswordResetEmail template for password reset requests
	PasswordResetEmail = EmailTemplate{
		Name:    "reset-password",
		Subject: "Reset Your Password",
	}

	// PasswordResetSuccessEmail template for successful password resets
	PasswordResetSuccessEmail = EmailTemplate{
		Name:    "reset-password-success",
		Subject: "Your Password Has Been Reset",
	}

	// ReminderEmail template for daily habit reminders
	ReminderEmail = EmailTemplate{
		Name:    "reminder",
		Subject: "Your Daily Habit Reminder",
	}

	// FirstHabitEmail template for users without habits
	FirstHabitEmail = EmailTemplate{
		Name:    "first-habit",
		Subject: "Start Your First Habit Today",
	}
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendTypedEmail(to string, template EmailTemplate, data interface{}) error
	SendWelcomeEmail(to, username string) error
	SendPasswordResetEmail(to, resetLink string, expiry time.Time) error
	SendPasswordResetSuccessEmail(to, username string) error
	SendReminderEmail(to string, firstName string, habits []HabitInfo, quote QuoteInfo) error
	SendFirstHabitEmail(to string, firstName string, quote QuoteInfo) error
	SendSimpleEmail(to, subject, content string) error
	GetCampaignManager() *CampaignManager
}
