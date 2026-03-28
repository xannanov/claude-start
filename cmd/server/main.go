package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

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
	fmt.Println("  add-user      - Добавить нового пользователя через CLI")
	fmt.Println("  list-users    - Показать всех пользователей")
	fmt.Println("  add-schedule  - Добавить расписание для существующего пользователя")
	fmt.Println("  run-scheduler - Запустить планировщик для периодической отправки писем")
	fmt.Println("  init-db       - Инициализировать базу данных")
	fmt.Println("  help          - Показать эту справку")
	fmt.Println("\nПример использования:")
	fmt.Println("  ./daily-email-sender add-user")
	fmt.Println("  ./daily-email-sender run-scheduler")
	fmt.Println()
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

	scheduler.Run(store, sender, 1*time.Minute)
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

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("ошибка чтения schema.sql: %w", err)
	}
	if err := store.ExecRaw(string(schema)); err != nil {
		return fmt.Errorf("ошибка выполнения схемы: %w", err)
	}

	slog.Info("база данных инициализирована")
	return nil
}
