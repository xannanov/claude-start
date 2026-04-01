package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// Config содержит все настройки приложения.
type Config struct {
	DatabaseURL string
	SMTP        SMTPConfig
	EmailFrom   string
	ServerPort  string // порт HTTP-сервера (по умолчанию 8080)
	SecretKey   []byte // ключ для подписи токенов (HMAC)
	AIAPIKey string // ключ Groq API (пустой = AI отключён)
	AIModel  string // модель AI (по умолчанию llama-3.3-70b-versatile)
	AIURL    string // базовый URL Groq API
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

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	secretKey, err := loadSecretKey()
	if err != nil {
		return nil, err
	}

	aiAPIKey := os.Getenv("AI_API_KEY")
	if aiAPIKey == "" {
		slog.Warn("AI_API_KEY не задан — AI-персонализация отключена, используются шаблоны")
	}

	aiModel := os.Getenv("AI_MODEL")
	if aiModel == "" {
		aiModel = "llama-3.3-70b-versatile"
	}

	aiURL := os.Getenv("AI_URL")
	if aiURL == "" {
		aiURL = "https://api.groq.com/openai/v1"
	}

	return &Config{
		DatabaseURL: dbURL,
		SMTP:        smtp,
		EmailFrom:   emailCfg.From,
		ServerPort:  port,
		SecretKey:   secretKey,
		AIAPIKey: aiAPIKey,
		AIModel:  aiModel,
		AIURL:    aiURL,
	}, nil
}

// loadSecretKey загружает SECRET_KEY из окружения или генерирует случайный.
func loadSecretKey() ([]byte, error) {
	keyHex := os.Getenv("SECRET_KEY")
	if keyHex != "" {
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("SECRET_KEY должен быть hex-строкой: %w", err)
		}
		if len(key) < 32 {
			return nil, fmt.Errorf("SECRET_KEY должен быть минимум 32 байта (64 hex-символа)")
		}
		return key, nil
	}

	slog.Warn("SECRET_KEY не задан — сгенерирован случайный (токены отписки не переживут перезапуск)")
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("ошибка генерации SECRET_KEY: %w", err)
	}
	return key, nil
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
