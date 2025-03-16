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

To test the email templates (Welcome, Password Reset, Daily Reminder, etc.), you can run the tool from any directory:

```bash
# From the root directory
go run cmd/emailtest/main.go

# Or from the emailtest directory
cd cmd/emailtest
go run main.go
```

This will:
1. Automatically find the root directory and load the .env file
2. Prompt you for a recipient email address
3. Show a menu of available email templates
4. Based on your selection, gather necessary information
5. Send the email using the configured SMTP settings

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

## Notes

- For the Daily Habit Reminder email, the tool will try to find the user in the database based on the email address. If found, it will use their actual habits. If not found, it will use test data.
- The email templates are located in `ui/email/` directory.
- These tools are for testing purposes only and should not be used in production. 