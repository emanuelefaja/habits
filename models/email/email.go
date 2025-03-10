package email

import "time"

// EmailTemplate defines a specific email type
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
	ResetToken string
	Expiry     time.Time
	AppName    string
}

// Predefined Email Templates
var (
	WelcomeEmail = EmailTemplate{
		Name:    "welcome",
		Subject: "Welcome to Habits!",
	}

	PasswordResetEmail = EmailTemplate{
		Name:    "password-reset",
		Subject: "Reset Your Habits Password",
	}
)

// EmailService interface
type EmailService interface {
	SendTypedEmail(to string, template EmailTemplate, data interface{}) error
	SendWelcomeEmail(to, username string) error
	SendPasswordResetEmail(to, resetToken string, expiry time.Time) error
}
