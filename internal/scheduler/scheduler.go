package scheduler

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"daily-email-sender/internal/database"
	"daily-email-sender/internal/email"
)

var moscowTZ *time.Location

func init() {
	var err error
	moscowTZ, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Error("не удалось загрузить Europe/Moscow", "error", err)
		os.Exit(1)
	}
}

// Scheduler управляет периодической отправкой писем по расписаниям пользователей.
type Scheduler struct {
	store    *database.Store
	sender   *email.Sender
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// New создаёт планировщик.
func New(store *database.Store, sender *email.Sender, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    store,
		sender:   sender,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start запускает планировщик в горутине.
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		slog.Info("планировщик запущен", "interval", s.interval)

		// Сразу проверяем при старте
		s.checkAndSendEmails()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkAndSendEmails()
			case <-s.stopChan:
				slog.Info("планировщик остановлен")
				return
			}
		}
	}()
}

// Stop gracefully останавливает планировщик.
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) checkAndSendEmails() {
	now := time.Now().In(moscowTZ)
	// Go: 0=Sunday → конвертация в 0=Monday
	dayOfWeek := (int(now.Weekday()) + 6) % 7
	currentHour := now.Hour()
	currentMinute := now.Minute()

	slog.Info("проверка расписаний",
		"time", now.Format("2006-01-02 15:04:05"),
		"day", dayOfWeek,
		"hour", currentHour,
		"minute", currentMinute,
	)

	schedules, err := s.store.GetActiveSchedulesForDay(dayOfWeek)
	if err != nil {
		slog.Error("ошибка получения расписаний", "error", err)
		return
	}

	for _, sc := range schedules {
		if sc.TimeHour == currentHour && sc.TimeMinute == currentMinute {
			s.sendEmailForSchedule(sc.ID, sc.UserID, sc.DayOfWeek, sc.TimeHour, sc.TimeMinute, sc.EmailType)
		}
	}
}

func (s *Scheduler) sendEmailForSchedule(id int, userID string, dayOfWeek, hour, minute int, emailType string) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		slog.Error("пользователь не найден", "schedule_id", id, "error", err)
		return
	}

	msg := email.GeneratePersonalizedMessage(*user, dayOfWeek, emailType)

	delays := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}
	for attempt, delay := range delays {
		err = s.sender.Send(user.Email, msg)
		if err == nil {
			slog.Info("письмо отправлено",
				"email", user.Email,
				"time", fmt.Sprintf("%02d:%02d", hour, minute),
				"type", emailType,
			)
			return
		}
		slog.Warn("ошибка отправки SMTP",
			"attempt", attempt+1,
			"email", user.Email,
			"error", err,
		)
		if attempt < len(delays)-1 {
			time.Sleep(delay)
		}
	}
	slog.Error("все попытки SMTP исчерпаны", "email", user.Email, "schedule_id", id)
}

// Run запускает планировщик, ждёт сигнала завершения, затем останавливает.
func Run(store *database.Store, sender *email.Sender, interval time.Duration) {
	s := New(store, sender, interval)
	s.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("получен сигнал завершения")
	s.Stop()
	store.Close()
	slog.Info("приложение завершено")
}
