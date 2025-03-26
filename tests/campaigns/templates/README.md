# Email Testing Tools

This directory contains tools for testing the email functionality of the Habits application.

## Prerequisites

Make sure your `.env` file is properly configured with SMTP settings in the root directory of your project:

```
SMTP_HOST=your-smtp-host
SMTP_USERNAME=your-username
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=your-from-email
SMTP_FROM_NAME=Your Name
```

The tools will automatically locate the `.env` file, database, and email templates by searching for the project root directory.

## Testing Email Templates

To test the email templates (Welcome, Password Reset, Daily Reminder, Campaign Emails, etc.), you can run the tool from any directory:

```bash
# From the root directory
go run cmd/emailtest/template_tester.go

# Or from the emailtest directory
cd cmd/emailtest
go run template_tester.go
```

This will:
1. Automatically find the root directory and load the .env file
2. Temporarily override APP_ENV to "production" to allow actual sending of emails
3. Prompt you for a recipient email address
4. Show a menu of available email templates
5. Based on your selection, gather necessary information
6. Send the email using the configured SMTP settings

## Testing Simple Emails

To send a simple custom email without using templates:

```bash
# From the root directory
go run cmd/emailtest/simple.go

# Or from the emailtest directory
cd cmd/emailtest
go run simple.go
```

This will:
1. Automatically find the root directory and load the .env file
2. Prompt you for a recipient email address
3. Ask for a subject and content
4. Send the email using the configured SMTP settings

## Testing Scheduler

To test the email scheduler:

```bash
# From the root directory
go run cmd/emailtest/schedulertest.go

# Or from the emailtest directory
cd cmd/emailtest
go run schedulertest.go
```

## Notes

- For the Daily Habit Reminder email, the tool will try to find the user in the database based on the email address. If found, it will use their actual habits. If not found, it will use test data.
- **IMPORTANT**: The template_tester.go tool temporarily sets APP_ENV to "production" to bypass the safeguard that prevents sending emails in development. This means real emails will be sent to the recipient address.
- The email templates are located in `ui/email/` directory.
- These tools are for testing purposes only and should not be used in production environments with real user data. 