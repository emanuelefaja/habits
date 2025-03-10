package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/wneessen/go-mail"
)

func main() {
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

	// Create new mail client
	client, err := mail.NewClient("smtp.emailit.com",
		mail.WithPort(587),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername("emailit"),
		mail.WithPassword("em_smtp_z4ytLekEHADz5ozHpY89OlM3UoRDAXLQ"),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Create new message
	msg := mail.NewMsg()
	if err := msg.From("Habits Team <noreply@mail.habits.co>"); err != nil {
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
