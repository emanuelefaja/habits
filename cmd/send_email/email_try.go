package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/wneessen/go-mail"
)

func main() {
	// Load .env file from project root
	projectRoot := filepath.Join("..", "..")
	if err := godotenv.Load(filepath.Join(projectRoot, ".env")); err != nil {
		fmt.Printf("Warning: .env file not found: %v\n", err)
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

	// Validate required environment variables
	if smtpHost == "" || smtpUsername == "" || smtpPassword == "" || fromEmail == "" || fromName == "" {
		fmt.Println("Error: Missing required SMTP settings in .env file")
		fmt.Println("Please ensure the following variables are set:")
		fmt.Println("- SMTP_HOST")
		fmt.Println("- SMTP_USERNAME")
		fmt.Println("- SMTP_PASSWORD")
		fmt.Println("- SMTP_FROM_EMAIL")
		fmt.Println("- SMTP_FROM_NAME")
		return
	}

	// Create new mail client
	client, err := mail.NewClient(smtpHost,
		mail.WithPort(587),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(smtpUsername),
		mail.WithPassword(smtpPassword),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Create new message
	msg := mail.NewMsg()
	if err := msg.From(fmt.Sprintf("%s <%s>", fromName, fromEmail)); err != nil {
		fmt.Printf("Error setting from address: %v\n", err)
		return
	}
	if err := msg.To(to); err != nil {
		fmt.Printf("Error setting to address: %v\n", err)
		return
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, content)

	// Send the email
	fmt.Println("Attempting to send email...")
	if err := client.DialAndSend(msg); err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}

	fmt.Println("Email sent successfully!")
}
