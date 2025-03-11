package email_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/wneessen/go-mail"
)

// TestSendEmail is a test function to send an email
func TestSendEmail(t *testing.T) {
	// Skip this test by default
	t.Skip("This test sends a real email. Run with -test.run=TestSendEmail to execute.")

	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	// Get user input
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter recipient email: ")
	to, _ := reader.ReadString('\n')
	to = strings.TrimSpace(to)

	fmt.Print("Enter email subject: ")
	subject, _ := reader.ReadString('\n')
	subject = strings.TrimSpace(subject)

	fmt.Print("Enter email content: ")
	content, _ := reader.ReadString('\n')
	content = strings.TrimSpace(content)

	// Get SMTP settings from environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	fromName := os.Getenv("SMTP_FROM_NAME")

	// Create new mail client
	client, err := mail.NewClient(smtpHost,
		mail.WithPort(587),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(smtpUsername),
		mail.WithPassword(smtpPassword),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	// Create new message
	msg := mail.NewMsg()
	if err := msg.From(fmt.Sprintf("%s <%s>", fromName, fromEmail)); err != nil {
		t.Fatalf("Error setting from address: %v", err)
	}
	if err := msg.To(to); err != nil {
		t.Fatalf("Error setting to address: %v", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, content)

	// Send the email
	fmt.Println("Attempting to send email...")
	if err := client.DialAndSend(msg); err != nil {
		t.Fatalf("Error sending email: %v", err)
	}

	fmt.Println("Email sent successfully!")
}
