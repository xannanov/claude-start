package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"daily-email-sender/internal/ai"
	"daily-email-sender/internal/database"
	"daily-email-sender/internal/email"
	"daily-email-sender/internal/models"
)

const maxConsecutiveErrors = 10

// taskQueueSize — максимальный размер буфера очереди AI-задач.
const taskQueueSize = 100

// defaultTaskInterval — интервал между задачами в очереди (защита от rate limit API).
const defaultTaskInterval = 8 * time.Second

var moscowTZ *time.Location

func init() {
	var err error
	moscowTZ, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Error("не удалось загрузить Europe/Moscow", "error", err)
		os.Exit(1)
	}
}

// emailTask — задача на отправку письма одному пользователю.
type emailTask struct {
	scheduleID int
	userID     string
	dayOfWeek  int
	hour       int
	minute     int
	emailType  string
}

// Scheduler управляет периодической отправкой писем по расписаниям пользователей.
type Scheduler struct {
	store             *database.Store
	sender            *email.Sender
	aiGenerator       *ai.Generator
	interval          time.Duration
	taskInterval      time.Duration
	taskQueue         chan emailTask
	stopChan          chan struct{}
	wg                sync.WaitGroup
	consecutiveErrors int
}

// New создаёт планировщик. aiGen может быть nil (AI отключён).
func New(store *database.Store, sender *email.Sender, aiGen *ai.Generator, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:        store,
		sender:       sender,
		aiGenerator:  aiGen,
		interval:     interval,
		taskInterval: defaultTaskInterval,
		taskQueue:    make(chan emailTask, taskQueueSize),
		stopChan:     make(chan struct{}),
	}
}

// Start запускает планировщик в горутине.
func (s *Scheduler) Start() {
	s.wg.Add(2)

	// Горутина обработки очереди
	go func() {
		defer s.wg.Done()
		s.processQueue()
	}()

	// Горутина проверки расписаний
	go func() {
		defer s.wg.Done()

		slog.Info("планировщик запущен", "interval", s.interval, "task_interval", s.taskInterval)

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

// GoWeekdayToISO конвертирует Go Weekday (0=Sunday, 1=Monday, ..., 6=Saturday)
// в ISO формат (0=Monday, 1=Tuesday, ..., 6=Sunday).
func GoWeekdayToISO(goWeekday time.Weekday) int {
	return (int(goWeekday) + 6) % 7
}

// checkAndSendEmails проверяет расписания и добавляет задачи в очередь.
func (s *Scheduler) checkAndSendEmails() {
	now := time.Now().In(moscowTZ)
	dayOfWeek := GoWeekdayToISO(now.Weekday())
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
		s.consecutiveErrors++
		slog.Warn("ошибка получения расписаний, повтор на следующем тике",
			"error", err,
			"consecutive_errors", s.consecutiveErrors,
		)
		if s.consecutiveErrors > maxConsecutiveErrors {
			slog.Error("превышен лимит последовательных ошибок, завершение работы",
				"consecutive_errors", s.consecutiveErrors,
			)
			os.Exit(1)
		}
		return
	}
	s.consecutiveErrors = 0

	for _, sc := range schedules {
		if sc.TimeHour == currentHour && sc.TimeMinute == currentMinute {
			task := emailTask{
				scheduleID: sc.ID,
				userID:     sc.UserID,
				dayOfWeek:  sc.DayOfWeek,
				hour:       sc.TimeHour,
				minute:     sc.TimeMinute,
				emailType:  sc.EmailType,
			}
			select {
			case s.taskQueue <- task:
				slog.Info("задача добавлена в очередь", "user_id", sc.UserID, "schedule_id", sc.ID)
			default:
				slog.Warn("очередь задач переполнена, пропуск", "user_id", sc.UserID, "schedule_id", sc.ID)
			}
		}
	}
}

// processQueue обрабатывает задачи из очереди с интервалом taskInterval.
func (s *Scheduler) processQueue() {
	for {
		select {
		case task := <-s.taskQueue:
			s.sendEmailForSchedule(task.scheduleID, task.userID, task.dayOfWeek, task.hour, task.minute, task.emailType)
			// Ждём перед следующей задачей (rate limit защита)
			select {
			case <-time.After(s.taskInterval):
			case <-s.stopChan:
				return
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *Scheduler) sendEmailForSchedule(id int, userID string, dayOfWeek, hour, minute int, emailType string) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		slog.Error("пользователь не найден", "schedule_id", id, "error", err)
		return
	}

	var msg models.PersonalizedMessage
	if s.aiGenerator != nil {
		msg = s.aiGenerator.GeneratePersonalizedMessage(*user, dayOfWeek, emailType)
	} else {
		msg = email.GeneratePersonalizedMessage(*user, dayOfWeek, emailType)
	}

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

// Run запускает планировщик и ждёт завершения через контекст или системный сигнал.
// Если ctx == context.Background(), ожидает SIGINT/SIGTERM (автономный режим).
func Run(store *database.Store, sender *email.Sender, aiGen *ai.Generator, interval time.Duration, ctx context.Context) {
	s := New(store, sender, aiGen, interval)
	s.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		slog.Info("получен сигнал завершения")
	case <-ctx.Done():
		slog.Info("планировщик получил команду остановки")
	}

	signal.Stop(sigChan)
	s.Stop()
	slog.Info("планировщик завершён")
}
