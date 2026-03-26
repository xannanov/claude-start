package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
)

// SMTPConfig stores SMTP server credentials
type SMTPConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
}

// EmailConfig stores email sender and recipient info
type EmailConfig struct {
	From string `json:"From"`
	To   string `json:"To"`
}

// Message represents an email message
type Message struct {
	Subject string
	Body    string
	Time    string
}


// moscowTZ holds the Moscow timezone location (UTC+3), loaded once at startup.
var moscowTZ *time.Location

func init() {
	var err error
	moscowTZ, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("failed to load Moscow timezone: %v", err)
	}
}

// Scheduler manages periodic email sending based on user schedules
type Scheduler struct {
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	isRunning   bool
}

// NewScheduler creates a new scheduler instance
func NewScheduler(interval time.Duration) *Scheduler {
	return &Scheduler{
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	if s.isRunning {
		log.Println("Scheduler is already running")
		return
	}

	s.isRunning = true
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		log.Printf("========================================")
		log.Printf("Scheduler started. Checking every %v", s.interval)
		log.Printf("========================================\n")

		// Run immediately on start
		log.Printf("Running initial check at %s", time.Now().Format("2006-01-02 15:04:05"))
		s.checkAndSendEmails()

		// Show next scheduled runs
		s.displayNextRuns()

		log.Printf("Next check scheduled in %v\n", s.interval)
		log.Printf("========================================\n")

		// Then run periodically
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkAndSendEmails()
			case <-s.stopChan:
				log.Println("\nScheduler stopping...")
				return
			}
		}
	}()
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.wg.Wait()
	s.isRunning = false
	log.Println("Scheduler stopped")
}

// checkAndSendEmails checks if any emails should be sent based on schedules
func (s *Scheduler) checkAndSendEmails() {
	// Get current Moscow time
	now := time.Now().In(moscowTZ)

	// Get day of week: Go Weekday() returns 0=Sun, we need 0=Mon
	dayOfWeek := (int(now.Weekday()) + 6) % 7

	// Get current hour and minute (Moscow)
	currentHour := now.Hour()
	currentMinute := now.Minute()

	log.Printf("Checking at %s (Day %d, %02d:%02d)...", now.Format("2006-01-02 15:04:05"), dayOfWeek, currentHour, currentMinute)

	// Get all active schedules for today
	schedules, err := GetActiveSchedulesForDay(dayOfWeek)
	if err != nil {
		log.Printf("Error getting schedules: %v", err)
		return
	}

	// Filter schedules that match current time
	var matchedSchedules []UserSchedule
	for _, schedule := range schedules {
		if schedule.TimeHour == currentHour && schedule.TimeMinute == currentMinute {
			matchedSchedules = append(matchedSchedules, schedule)
		}
	}

	if len(matchedSchedules) > 0 {
		log.Printf("  Found %d email(s) to send:", len(matchedSchedules))
		for _, schedule := range matchedSchedules {
			log.Printf("    - %s at %02d:%02d (%s)", schedule.UserID, schedule.TimeHour, schedule.TimeMinute, schedule.EmailType)
		}
	} else {
		log.Printf("  No emails scheduled at this time")
	}

	// Send emails for matched schedules
	for _, schedule := range matchedSchedules {
		s.sendEmailForSchedule(schedule)
	}
}

// sendEmailForSchedule sends an email for a specific schedule
func (s *Scheduler) sendEmailForSchedule(schedule UserSchedule) {
	// Get user
	user, err := GetUserByID(schedule.UserID)
	if err != nil {
		log.Printf("Error getting user for schedule %d: %v", schedule.ID, err)
		return
	}

	// Generate personalized message
	message := GeneratePersonalizedMessage(*user, schedule.DayOfWeek, schedule.EmailType)

	// Send email using SMTP config from .env
	err = sendEmailFromConfig(message, user.Email)
	if err != nil {
		log.Printf("Error sending email to %s: %v", user.Email, err)
		return
	}

	log.Printf("Email sent to %s at %d:%02d (%s)", user.Email,
		message.TimeOfDay, schedule.TimeHour, schedule.EmailType)
}

// getNextRuns returns schedules that will run within the next specified minutes
func getNextRuns(hoursAhead int) ([]UserSchedule, error) {
	now := time.Now().In(moscowTZ)
	dayOfWeek := (int(now.Weekday()) + 6) % 7
	nextDay := now.AddDate(0, 0, 1)

	var allSchedules []UserSchedule

	// Get schedules for today
	schedules, err := GetActiveSchedulesForDay(dayOfWeek)
	if err != nil {
		return nil, err
	}
	allSchedules = append(allSchedules, schedules...)

	// Get schedules for tomorrow if within range
	if hoursAhead >= 24 {
		schedulesTomorrow, err := GetActiveSchedulesForDay((dayOfWeek + 1) % 7)
		if err != nil {
			return nil, err
		}
		allSchedules = append(allSchedules, schedulesTomorrow...)
	}

	// Filter schedules by time range
	var withinRange []UserSchedule
	for _, schedule := range allSchedules {
		scheduleTime := time.Date(now.Year(), now.Month(), now.Day(), schedule.TimeHour, schedule.TimeMinute, 0, 0, moscowTZ)
		if scheduleTime.After(now) && scheduleTime.Before(nextDay) {
			withinRange = append(withinRange, schedule)
		}
	}

	// Sort by time
	sort.Slice(withinRange, func(i, j int) bool {
		if withinRange[i].TimeHour != withinRange[j].TimeHour {
			return withinRange[i].TimeHour < withinRange[j].TimeHour
		}
		return withinRange[i].TimeMinute < withinRange[j].TimeMinute
	})

	return withinRange, nil
}

// displayNextRuns shows the next scheduled email runs
func (s *Scheduler) displayNextRuns() {
	log.Printf("Next scheduled emails:")
	now := time.Now().In(moscowTZ)
	dayOfWeek := (int(now.Weekday()) + 6) % 7
	dayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	log.Printf("Current time: %s, Day: %d (%s) (0=Mon, 6=Sun)", now.Format("2006-01-02 15:04:05"), dayOfWeek, dayNames[dayOfWeek])

	// Get all active schedules for today
	schedules, err := GetActiveSchedulesForDay(dayOfWeek)
	if err != nil {
		log.Printf("  Error getting schedules: %v", err)
		return
	}

	log.Printf("Found %d schedules for day %d:", len(schedules), dayOfWeek)
	for _, schedule := range schedules {
		log.Printf("  - Schedule ID: %d, UserID: %s, Time: %02d:%02d, Type: %s",
			schedule.ID, schedule.UserID, schedule.TimeHour, schedule.TimeMinute, schedule.EmailType)
	}

	if len(schedules) == 0 {
		log.Printf("  No emails scheduled for today")
		return
	}

	// Get schedules within 24 hours
	schedules, err = getNextRuns(24)
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	if len(schedules) == 0 {
		log.Printf("  No emails scheduled in the next 24 hours")
		return
	}

	for _, schedule := range schedules {
		// Calculate time until this schedule
		var daysUntil int
		if schedule.TimeHour > now.Hour() || (schedule.TimeHour == now.Hour() && schedule.TimeMinute > now.Minute()) {
			// Время ещё не наступило, может быть сегодня или позже
			daysUntil = schedule.DayOfWeek - dayOfWeek
			if daysUntil < 0 {
				daysUntil += 7
			}
		} else {
			// Время уже прошло, значит расписание на следующей неделе
			daysUntil = (7 + schedule.DayOfWeek - dayOfWeek) % 7
		}

		var timeStr string
		if daysUntil > 0 {
			timeStr = fmt.Sprintf("in %d day(s) at %02d:%02d", daysUntil, schedule.TimeHour, schedule.TimeMinute)
		} else {
			minutesUntil := schedule.TimeMinute - now.Minute()
			if minutesUntil < 0 {
				minutesUntil += 60
			}
			timeStr = fmt.Sprintf("in %d minute(s) at %02d:%02d", minutesUntil, schedule.TimeHour, schedule.TimeMinute)
		}

		log.Printf("  - %s: %s (%s)", schedule.UserID, timeStr, schedule.EmailType)
	}
}

// sendEmailFromConfig sends email using SMTP configuration from environment
func sendEmailFromConfig(message PersonalizedMessage, toEmail string) error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Parse SMTP config from env
	smtpConfigStr := os.Getenv("SMTP_CONFIG")
	var smtpConfig SMTPConfig
	if err := json.Unmarshal([]byte(smtpConfigStr), &smtpConfig); err != nil {
		return fmt.Errorf("failed to parse SMTP config: %w", err)
	}

	// Parse email config from env
	emailConfigStr := os.Getenv("EMAIL_CONFIG")
	var emailConfig EmailConfig
	if err := json.Unmarshal([]byte(emailConfigStr), &emailConfig); err != nil {
		return fmt.Errorf("failed to parse email config: %w", err)
	}

	// Send email directly
	return sendEmail(smtpConfig, EmailConfig{From: emailConfig.From, To: toEmail}, Message{
		Subject: message.Subject,
		Body:    message.Body,
		Time:    message.TimeOfDay,
	})
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RunScheduler starts the scheduler with default 1-hour interval
func RunScheduler(interval time.Duration) {
	scheduler := NewScheduler(interval)
	scheduler.Start()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("\nReceived shutdown signal")
	scheduler.Stop()
	CloseDatabase()
	log.Println("Application shutting down")
}

// sendEmail sends email via SMTP
func sendEmail(smtp SMTPConfig, email EmailConfig, message Message) error {
	m := gomail.NewMessage()
	m.SetHeader("From", email.From)
	m.SetHeader("To", email.To)
	m.SetHeader("Subject", message.Subject)
	m.SetBody("text/html", message.Body)

	d := gomail.NewDialer(smtp.Host, smtp.Port, smtp.User, smtp.Password)
	d.SSL = true
	d.TLSConfig = &tls.Config{ServerName: smtp.Host}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully at %s: %s", message.Time, message.Subject)
	return nil
}

