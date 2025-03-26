package main

import (
	"bufio"
	"fmt"
	"mad/models/email"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// This is a utility for sending simple emails
// To use this file, run: go run tests/email/simple.go
func main() {
	fmt.Println("📧 Simple Email Sender")
	fmt.Println("=====================")

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

	// Send a simple email
	err = emailService.SendSimpleEmail(to, subject, content)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}

	fmt.Println("✅ Email sent successfully!")
}

// findRootDir attempts to find the root directory of the project
func findRootDir() (string, error) {
	// Try to find the root directory by looking for common files
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check current directory first
	if _, err := os.Stat(filepath.Join(currentDir, ".env")); err == nil {
		return currentDir, nil
	}

	// Try parent directory (for tests/email)
	rootDir := filepath.Join(currentDir, "..")
	if _, err := os.Stat(filepath.Join(rootDir, ".env")); err == nil {
		return rootDir, nil
	}

	// Try parent of parent directory (for project root)
	rootDir = filepath.Join(currentDir, "../..")
	if _, err := os.Stat(filepath.Join(rootDir, ".env")); err == nil {
		return rootDir, nil
	}

	// If still not found, return an error
	return "", fmt.Errorf("could not find root directory with .env file")
}
