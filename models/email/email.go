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
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendTypedEmail(to string, template EmailTemplate, data interface{}) error
	SendWelcomeEmail(to, username string) error
	SendPasswordResetEmail(to, resetLink string, expiry time.Time) error
	SendPasswordResetSuccessEmail(to, username string) error
}
