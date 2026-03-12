package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "add-user":
		// Add user via CLI
		cli := NewCLI()
		if err := cli.AddUserInteractive(); err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

	case "list-users":
		// List all users
		cli := NewCLI()
		if err := cli.ListUsers(); err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

	case "add-schedule":
		// Add schedule for existing user
		cli := NewCLI()
		if err := cli.AddScheduleInteractive(); err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

	case "run-scheduler":
		// Run the scheduler to send emails periodically
		if err := runScheduler(); err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

	case "init-db":
		// Initialize database schema
		if err := initDatabase(); err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("\n=== Daily Email Sender ===")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  add-user      - Добавить нового пользователя через CLI")
	fmt.Println("  list-users    - Показать всех пользователей")
	fmt.Println("  add-schedule  - Добавить расписание для существующего пользователя")
	fmt.Println("  run-scheduler - Запустить планировщик для периодической отправки писем")
	fmt.Println("  init-db       - Инициализировать базу данных")
	fmt.Println("  help          - Показать эту справку")
	fmt.Println("\nExample usage:")
	fmt.Println("  go run main.go add-user")
	fmt.Println("  go run main.go run-scheduler")
	fmt.Println()
}

func runScheduler() error {
	// Connect to database
	if err := ConnectToDatabase(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer CloseDatabase()

	// Run scheduler with 1-minute interval
	RunScheduler(1 * time.Minute)
	return nil
}

func initDatabase() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Connect to database with UTF-8 encoding
	connStr := getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Read SQL schema
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// Execute schema
	if _, err = db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}
