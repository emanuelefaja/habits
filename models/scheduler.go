package models

import (
	"database/sql"
	"log"
	"os"
	"time"

	"mad/models/email"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled tasks for the application
type Scheduler struct {
	db         *sql.DB
	emailSvc   email.EmailService
	cron       *cron.Cron
	batchSize  int
	batchDelay time.Duration
	dailyTime  string
	weeklyTime string
	weeklyDay  time.Weekday
	isRunning  bool
	stopChan   chan struct{}
}

// NewScheduler creates a new scheduler with the given database and email service
func NewScheduler(db *sql.DB, emailSvc email.EmailService) *Scheduler {
	return &Scheduler{
		db:         db,
		emailSvc:   emailSvc,
		cron:       cron.New(),
		batchSize:  25,                     // Default batch size of 25 emails
		batchDelay: 200 * time.Millisecond, // Default delay of 200ms between batches
		dailyTime:  "0 19 * * *",           // Default to 7 PM daily
		weeklyTime: "0 18 * * 0",           // Default to 6 PM on Sundays
		weeklyDay:  time.Sunday,
		isRunning:  false,
		stopChan:   make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	if s.isRunning {
		return nil // Already running
	}

	// Schedule daily reminders for users with habits
	_, err := s.cron.AddFunc(s.dailyTime, func() {
		s.sendDailyReminders()
	})
	if err != nil {
		return err
	}

	// Schedule weekly nudges for users without habits
	_, err = s.cron.AddFunc(s.weeklyTime, func() {
		s.sendWeeklyFirstHabitReminders()
	})
	if err != nil {
		return err
	}

	// Schedule campaign email sending (every minute with rate limiting)
	_, err = s.cron.AddFunc("* * * * *", func() {
		s.SendCampaignEmailsBatch(20) // Send 20 emails per minute (1200/hour max)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("Scheduler started successfully")
	return nil
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}
	s.cron.Stop()
	s.isRunning = false
	log.Println("Scheduler stopped")
}

// SetBatchSize sets the number of emails to send in each batch
func (s *Scheduler) SetBatchSize(size int) {
	if size < 1 {
		size = 1
	}
	s.batchSize = size
}

// SetBatchDelay sets the delay between batches
func (s *Scheduler) SetBatchDelay(delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	s.batchDelay = delay
}

// SetDailyReminderTime sets the time for daily reminders (cron format)
func (s *Scheduler) SetDailyReminderTime(cronExpr string) error {
	// Validate cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return err
	}
	s.dailyTime = cronExpr
	return nil
}

// SetWeeklyReminderTime sets the time for weekly reminders (cron format)
func (s *Scheduler) SetWeeklyReminderTime(cronExpr string) error {
	// Validate cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return err
	}
	s.weeklyTime = cronExpr
	return nil
}

// sendDailyReminders sends reminder emails to users with habits
func (s *Scheduler) sendDailyReminders() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // Default to development if not set
	}

	users, err := GetUsersWithHabitsAndNotificationsEnabled(s.db)
	if err != nil {
		log.Printf("Error getting users with habits: %v", err)
		return
	}

	log.Printf("[%s] Sending daily reminders to %d users", env, len(users))

	// If not in production, log more details but don't actually send emails
	// (The email service will handle skipping sends, this is just for better logging)
	if env != "production" {
		log.Printf("[%s] Email sending is disabled in non-production environments", env)
		for _, user := range users {
			log.Printf("[%s] Would send reminder to: %s (%s)", env, user.Email, user.FirstName)
		}
		return
	}

	// Process users in batches
	for i := 0; i < len(users); i += s.batchSize {
		end := i + s.batchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		s.processDailyReminderBatch(batch)

		// Sleep between batches to avoid overwhelming the SMTP server
		if end < len(users) {
			time.Sleep(s.batchDelay)
		}
	}
}

// processDailyReminderBatch processes a batch of users for daily reminders
func (s *Scheduler) processDailyReminderBatch(users []*User) {
	for _, user := range users {
		// Get the user's habits
		habits, err := GetHabitsByUserID(s.db, int(user.ID))
		if err != nil {
			log.Printf("Error getting habits for user %d: %v", user.ID, err)
			continue
		}

		// Skip if no habits (shouldn't happen due to our query, but just in case)
		if len(habits) == 0 {
			continue
		}

		// Convert habits to email format
		habitInfos := make([]email.HabitInfo, 0, len(habits))
		for _, habit := range habits {
			habitInfos = append(habitInfos, email.HabitInfo{
				Name:  habit.Name,
				Emoji: habit.Emoji,
			})
		}

		// Get a random quote
		quote, err := GetRandomQuoteForEmail()
		if err != nil {
			log.Printf("Error getting quote: %v", err)
			// Continue with empty quote rather than failing
			quote = email.QuoteInfo{
				Text:   "Success is the sum of small efforts, repeated day in and day out.",
				Author: "Robert Collier",
			}
		}

		// Send the email
		err = s.emailSvc.SendReminderEmail(user.Email, user.FirstName, habitInfos, quote)
		if err != nil {
			log.Printf("Error sending reminder email to %s: %v", user.Email, err)
			// Continue with next user rather than failing the whole batch
		} else {
			log.Printf("Sent reminder email to %s", user.Email)
		}
	}
}

// sendWeeklyFirstHabitReminders sends emails to users without habits
func (s *Scheduler) sendWeeklyFirstHabitReminders() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // Default to development if not set
	}

	users, err := GetUsersWithNoHabitsAndNotificationsEnabled(s.db)
	if err != nil {
		log.Printf("Error getting users without habits: %v", err)
		return
	}

	log.Printf("[%s] Sending first habit reminders to %d users", env, len(users))

	// If not in production, log more details but don't actually send emails
	if env != "production" {
		log.Printf("[%s] Email sending is disabled in non-production environments", env)
		for _, user := range users {
			log.Printf("[%s] Would send first habit reminder to: %s (%s)", env, user.Email, user.FirstName)
		}
		return
	}

	// Process users in batches
	for i := 0; i < len(users); i += s.batchSize {
		end := i + s.batchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		s.processFirstHabitReminderBatch(batch)

		// Sleep between batches to avoid overwhelming the SMTP server
		if end < len(users) {
			time.Sleep(s.batchDelay)
		}
	}
}

// processFirstHabitReminderBatch processes a batch of users for first habit reminders
func (s *Scheduler) processFirstHabitReminderBatch(users []*User) {
	for _, user := range users {
		// Get a random quote
		quote, err := GetRandomQuoteForEmail()
		if err != nil {
			log.Printf("Error getting quote: %v", err)
			// Continue with empty quote rather than failing
			quote = email.QuoteInfo{
				Text:   "The secret of getting ahead is getting started.",
				Author: "Mark Twain",
			}
		}

		// Send the email
		err = s.emailSvc.SendFirstHabitEmail(user.Email, user.FirstName, quote)
		if err != nil {
			log.Printf("Error sending first habit email to %s: %v", user.Email, err)
			// Continue with next user rather than failing the whole batch
		} else {
			log.Printf("Sent first habit email to %s", user.Email)
		}
	}
}

// RunDailyRemindersNow triggers the daily reminder job immediately
func (s *Scheduler) RunDailyRemindersNow() {
	go s.sendDailyReminders()
}

// RunWeeklyFirstHabitRemindersNow triggers the weekly first habit reminder job immediately
func (s *Scheduler) RunWeeklyFirstHabitRemindersNow() {
	go s.sendWeeklyFirstHabitReminders()
}

// SendCampaignEmails sends pending campaign emails (legacy method, uses default batch size)
func (s *Scheduler) SendCampaignEmails() {
	log.Println("🔔 Running scheduled campaign email sending")

	campaignManager := email.NewCampaignManager(s.db, s.emailSvc)
	if err := campaignManager.SendPendingCampaignEmails(); err != nil {
		log.Printf("❌ Error sending campaign emails: %v", err)
	}
}

// SendCampaignEmailsBatch sends pending campaign emails with a specified batch size
func (s *Scheduler) SendCampaignEmailsBatch(batchSize int) {
	log.Printf("🔔 Running scheduled campaign email sending (batch size: %d)", batchSize)

	campaignManager := email.NewCampaignManager(s.db, s.emailSvc)
	if err := campaignManager.SendPendingCampaignEmailsWithLimit(batchSize); err != nil {
		log.Printf("❌ Error sending campaign emails: %v", err)
	}
}
