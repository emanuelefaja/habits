package email

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wneessen/go-mail"
)

// SMTPConfig holds configuration for SMTP connections
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromName    string
	FromEmail   string
	TemplateDir string
	Secure      bool
	RequireTLS  bool
	MaxConns    int
}

// SMTPEmailService implements EmailService
type SMTPEmailService struct {
	config          SMTPConfig
	client          *mail.Client
	campaignManager *CampaignManager
}

// NewSMTPEmailService creates a new SMTP email service
func NewSMTPEmailService(config SMTPConfig) (EmailService, error) {
	client, err := mail.NewClient(config.Host,
		mail.WithPort(config.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create mail client: %w", err)
	}

	return &SMTPEmailService{
		config: config,
		client: client,
	}, nil
}

// SendTypedEmail sends an email using the specified template and data
func (s *SMTPEmailService) SendTypedEmail(to string, template EmailTemplate, data interface{}) error {
	// Check if we're in development/staging mode
	if env := os.Getenv("APP_ENV"); env != "production" {
		log.Printf("[%s MODE] Would send email to %s with subject: %s", env, to, template.Subject)
		return nil
	}

	// Load and render templates
	htmlContent, textContent, err := s.renderTemplates(template.Name, data)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	msg := mail.NewMsg()
	if err := msg.From(fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("failed to set to address: %w", err)
	}
	msg.Subject(template.Subject)

	msg.SetBodyString(mail.TypeTextPlain, textContent)
	msg.AddAlternativeString(mail.TypeTextHTML, htmlContent)

	return s.client.DialAndSend(msg)
}

// SendWelcomeEmail sends a welcome email to a new user
func (s *SMTPEmailService) SendWelcomeEmail(to, username string) error {
	data := WelcomeEmailData{
		Username: username,
		AppName:  "The Habits Company",
	}
	return s.SendTypedEmail(to, WelcomeEmail, data)
}

// SendPasswordResetEmail sends a password reset email
func (s *SMTPEmailService) SendPasswordResetEmail(to, resetLink string, expiry time.Time) error {
	log.Printf("📧 Preparing password reset email for: %s", to)

	data := PasswordResetEmailData{
		ResetLink:   resetLink,
		ExpiryHours: "1",     // Token expires in 1 hour
		Username:    "there", // Generic greeting since we don't have the username here
		AppName:     "The Habits Company",
	}
	log.Printf("📝 Email data prepared: %+v", data)

	err := s.SendTypedEmail(to, PasswordResetEmail, data)
	if err != nil {
		log.Printf("❌ Failed to send password reset email: %v", err)
		return err
	}

	log.Printf("✅ Password reset email sent successfully to: %s", to)
	return nil
}

// SendPasswordResetSuccessEmail sends a password reset success email
func (s *SMTPEmailService) SendPasswordResetSuccessEmail(to, username string) error {
	data := PasswordResetSuccessEmailData{
		Username:  username,
		AppName:   "The Habits Company",
		LoginLink: "https://habits.co/login",
	}
	return s.SendTypedEmail(to, PasswordResetSuccessEmail, data)
}

// SendReminderEmail sends a daily habit reminder email
func (s *SMTPEmailService) SendReminderEmail(to string, firstName string, habits []HabitInfo, quote QuoteInfo) error {
	data := ReminderEmailData{
		FirstName: firstName,
		Habits:    habits,
		Quote:     quote,
		AppName:   "The Habits Company",
	}
	return s.SendTypedEmail(to, ReminderEmail, data)
}

// SendFirstHabitEmail sends an email encouraging users to create their first habit
func (s *SMTPEmailService) SendFirstHabitEmail(to string, firstName string, quote QuoteInfo) error {
	data := FirstHabitEmailData{
		FirstName: firstName,
		Quote:     quote,
		AppName:   s.config.FromName,
	}
	return s.SendTypedEmail(to, FirstHabitEmail, data)
}

// SendSimpleEmail sends a simple email with custom subject and content
func (s *SMTPEmailService) SendSimpleEmail(to, subject, content string) error {
	// Create a custom template for the simple email
	customTemplate := EmailTemplate{
		Name:    "custom",
		Subject: subject,
	}

	// Send the email with the content as data
	return s.SendTypedEmail(to, customTemplate, map[string]string{
		"Content": content,
	})
}

// SetCampaignManager sets the campaign manager for the email service
func (s *SMTPEmailService) SetCampaignManager(cm *CampaignManager) {
	s.campaignManager = cm
}

// GetCampaignManager returns the campaign manager
func (s *SMTPEmailService) GetCampaignManager() *CampaignManager {
	return s.campaignManager
}
