package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // встраивает timezone DB в бинарник (нужно для Alpine/Docker без tzdata)

	"daily-email-sender/internal/ai"
	"daily-email-sender/internal/api"
	"daily-email-sender/internal/auth"
	"daily-email-sender/internal/cli"
	"daily-email-sender/internal/config"
	"daily-email-sender/internal/database"
	"daily-email-sender/internal/email"
	"daily-email-sender/internal/scheduler"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(); err != nil {
			slog.Error("ошибка сервера", "error", err)
			os.Exit(1)
		}

	case "add-user":
		if err := runCLI(func(c *cli.CLI) error { return c.AddUserInteractive() }); err != nil {
			slog.Error("ошибка", "error", err)
			os.Exit(1)
		}

	case "list-users":
		if err := runCLI(func(c *cli.CLI) error { return c.ListUsers() }); err != nil {
			slog.Error("ошибка", "error", err)
			os.Exit(1)
		}

	case "add-schedule":
		if err := runCLI(func(c *cli.CLI) error { return c.AddScheduleInteractive() }); err != nil {
			slog.Error("ошибка", "error", err)
			os.Exit(1)
		}

	case "run-scheduler":
		if err := runScheduler(); err != nil {
			slog.Error("ошибка планировщика", "error", err)
			os.Exit(1)
		}

	case "init-db":
		if err := initDatabase(); err != nil {
			slog.Error("ошибка инициализации БД", "error", err)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	default:
		slog.Error("неизвестная команда", "command", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("\n=== Daily Email Sender ===")
	fmt.Println("\nДоступные команды:")
	fmt.Println("  serve         - Запустить HTTP-сервер + планировщик")
	fmt.Println("  add-user      - Добавить нового пользователя через CLI")
	fmt.Println("  list-users    - Показать всех пользователей")
	fmt.Println("  add-schedule  - Добавить расписание для существующего пользователя")
	fmt.Println("  run-scheduler - Запустить только планировщик (без HTTP)")
	fmt.Println("  init-db       - Инициализировать базу данных")
	fmt.Println("  help          - Показать эту справку")
	fmt.Println("\nПример использования:")
	fmt.Println("  ./daily-email-sender serve")
	fmt.Println("  ./daily-email-sender add-user")
	fmt.Println()
}

// runServe запускает HTTP-сервер и планировщик одновременно.
func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("ошибка конфигурации: %w", err)
	}

	store, err := database.NewStore(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}
	defer store.Close()

	slog.Info("подключение к PostgreSQL установлено")

	sender := email.NewSender(cfg.SMTP, cfg.EmailFrom)
	slog.Info("проверка SMTP-соединения...")
	if err := sender.CheckConnection(); err != nil {
		return fmt.Errorf("SMTP недоступен: %w", err)
	}
	slog.Info("SMTP-соединение успешно проверено")

	sessions := auth.NewSessionManager()
	defer sessions.Stop()

	srv, err := api.NewServer(store, sessions, cfg.SecretKey, cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("ошибка создания сервера: %w", err)
	}

	// Инициализация AI-генератора (Groq API)
	var aiGen *ai.Generator
	if cfg.AIAPIKey != "" {
		groqClient := ai.NewGroqClient(cfg.AIAPIKey, cfg.AIURL, cfg.AIModel)
		aiGen = ai.NewGenerator(groqClient, store, cfg.AIModel)
		slog.Info("AI-персонализация включена", "model", cfg.AIModel)
	} else {
		slog.Warn("AI-персонализация отключена (AI_API_KEY не задан)")
	}

	// Запускаем scheduler в фоне
	schedulerDone := make(chan struct{})
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	defer schedulerCancel()
	go func() {
		defer close(schedulerDone)
		scheduler.Run(store, sender, aiGen, 1*time.Minute, schedulerCtx)
	}()

	// Запускаем HTTP-сервер в фоне
	httpErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
		close(httpErr)
	}()

	// Ожидаем сигнал завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("получен сигнал завершения", "signal", sig)
	case err := <-httpErr:
		if err != nil {
			schedulerCancel()
			<-schedulerDone
			return fmt.Errorf("HTTP-сервер завершился с ошибкой: %w", err)
		}
	}

	// Graceful shutdown
	slog.Info("завершение работы...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	schedulerCancel()
	<-schedulerDone

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("ошибка остановки HTTP-сервера: %w", err)
	}

	slog.Info("сервер остановлен")
	return nil
}

// runCLI подключается к БД только с DATABASE_URL, выполняет fn, закрывает соединение.
func runCLI(fn func(*cli.CLI) error) error {
	dbURL, err := config.LoadForCLI()
	if err != nil {
		return err
	}
	store, err := database.NewStore(dbURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}
	defer store.Close()

	slog.Info("подключение к PostgreSQL установлено")
	return fn(cli.New(store))
}

// runScheduler загружает полный конфиг и запускает планировщик.
func runScheduler() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("ошибка конфигурации: %w", err)
	}

	store, err := database.NewStore(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	slog.Info("подключение к PostgreSQL установлено")

	sender := email.NewSender(cfg.SMTP, cfg.EmailFrom)
	slog.Info("проверка SMTP-соединения...")
	if err := sender.CheckConnection(); err != nil {
		return fmt.Errorf("SMTP недоступен, scheduler не запущен: %w", err)
	}
	slog.Info("SMTP-соединение успешно проверено")

	var aiGen *ai.Generator
	if cfg.AIAPIKey != "" {
		groqClient := ai.NewGroqClient(cfg.AIAPIKey, cfg.AIURL, cfg.AIModel)
		aiGen = ai.NewGenerator(groqClient, store, cfg.AIModel)
		slog.Info("AI-персонализация включена", "model", cfg.AIModel)
	} else {
		slog.Warn("AI-персонализация отключена (AI_API_KEY не задан)")
	}

	scheduler.Run(store, sender, aiGen, 1*time.Minute, context.Background())
	return nil
}

// initDatabase инициализирует схему БД из schema.sql.
func initDatabase() error {
	dbURL, err := config.LoadForCLI()
	if err != nil {
		return err
	}
	store, err := database.NewStore(dbURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}
	defer store.Close()

	schema, err := os.ReadFile("migrations/schema.sql")
	if err != nil {
		return fmt.Errorf("ошибка чтения migrations/schema.sql: %w", err)
	}
	if err := store.ExecRaw(string(schema)); err != nil {
		return fmt.Errorf("ошибка выполнения схемы: %w", err)
	}

	slog.Info("база данных инициализирована")
	return nil
}
