package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config содержит все настройки приложения
type Config struct {
	DatabaseURL string
	SMTP        SMTPConfig
	EmailFrom   string
}

// SMTPConfig содержит параметры SMTP-сервера
type SMTPConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
}

// Load загружает конфигурацию из .env и переменных окружения.
// .env загружается один раз при старте.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// Не фатально — переменные могут быть заданы напрямую
		fmt.Fprintf(os.Stderr, "Предупреждение: .env файл не найден: %v\n", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL не задан")
	}

	smtpStr := os.Getenv("SMTP_CONFIG")
	if smtpStr == "" {
		return nil, fmt.Errorf("SMTP_CONFIG не задан")
	}
	var smtp SMTPConfig
	if err := json.Unmarshal([]byte(smtpStr), &smtp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга SMTP_CONFIG: %w", err)
	}
	if smtp.Host == "" || smtp.Port == 0 || smtp.User == "" || smtp.Password == "" {
		return nil, fmt.Errorf("SMTP_CONFIG неполный: нужны Host, Port, User, Password")
	}

	emailCfgStr := os.Getenv("EMAIL_CONFIG")
	if emailCfgStr == "" {
		return nil, fmt.Errorf("EMAIL_CONFIG не задан")
	}
	var emailCfg struct {
		From string `json:"From"`
	}
	if err := json.Unmarshal([]byte(emailCfgStr), &emailCfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга EMAIL_CONFIG: %w", err)
	}
	if emailCfg.From == "" {
		return nil, fmt.Errorf("EMAIL_CONFIG.From не задан")
	}

	return &Config{
		DatabaseURL: dbURL,
		SMTP:        smtp,
		EmailFrom:   emailCfg.From,
	}, nil
}

// LoadForCLI загружает только DATABASE_URL для CLI-команд (без SMTP).
func LoadForCLI() (string, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: .env файл не найден: %v\n", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8"
	}
	return dbURL, nil
}
