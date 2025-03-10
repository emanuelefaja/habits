package email

import (
	"fmt"
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
	config SMTPConfig
	client *mail.Client
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
		AppName:  "Habits",
	}
	return s.SendTypedEmail(to, WelcomeEmail, data)
}

// SendPasswordResetEmail sends a password reset email
func (s *SMTPEmailService) SendPasswordResetEmail(to, resetToken string, expiry time.Time) error {
	data := PasswordResetEmailData{
		ResetToken: resetToken,
		Expiry:     expiry,
		AppName:    "Habits",
	}
	return s.SendTypedEmail(to, PasswordResetEmail, data)
}
