package config

import (
	"os"
	"testing"
)

// setEnv устанавливает переменные окружения для теста и возвращает функцию очистки.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestLoad_Success(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		"SMTP_CONFIG":  `{"Host":"smtp.yandex.ru","Port":465,"User":"user@ya.ru","Password":"secret"}`,
		"EMAIL_CONFIG": `{"From":"noreply@ya.ru"}`,
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb?sslmode=disable" {
		t.Errorf("DatabaseURL=%q", cfg.DatabaseURL)
	}
	if cfg.SMTP.Host != "smtp.yandex.ru" {
		t.Errorf("SMTP.Host=%q", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != 465 {
		t.Errorf("SMTP.Port=%d", cfg.SMTP.Port)
	}
	if cfg.SMTP.User != "user@ya.ru" {
		t.Errorf("SMTP.User=%q", cfg.SMTP.User)
	}
	if cfg.SMTP.Password != "secret" {
		t.Errorf("SMTP.Password=%q", cfg.SMTP.Password)
	}
	if cfg.EmailFrom != "noreply@ya.ru" {
		t.Errorf("EmailFrom=%q", cfg.EmailFrom)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	// Убедимся что DATABASE_URL не задан
	os.Unsetenv("DATABASE_URL")
	setEnv(t, map[string]string{
		"SMTP_CONFIG":  `{"Host":"smtp.yandex.ru","Port":465,"User":"u","Password":"p"}`,
		"EMAIL_CONFIG": `{"From":"a@b.com"}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestLoad_MissingSMTPConfig(t *testing.T) {
	os.Unsetenv("SMTP_CONFIG")
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"EMAIL_CONFIG": `{"From":"a@b.com"}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing SMTP_CONFIG")
	}
}

func TestLoad_InvalidSMTPJSON(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SMTP_CONFIG":  `{invalid json}`,
		"EMAIL_CONFIG": `{"From":"a@b.com"}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid SMTP JSON")
	}
}

func TestLoad_IncompleteSMTP(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SMTP_CONFIG":  `{"Host":"smtp.yandex.ru","Port":465}`,
		"EMAIL_CONFIG": `{"From":"a@b.com"}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for incomplete SMTP config (missing User/Password)")
	}
}

func TestLoad_MissingEmailConfig(t *testing.T) {
	os.Unsetenv("EMAIL_CONFIG")
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SMTP_CONFIG":  `{"Host":"smtp.yandex.ru","Port":465,"User":"u","Password":"p"}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing EMAIL_CONFIG")
	}
}

func TestLoad_EmptyEmailFrom(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"SMTP_CONFIG":  `{"Host":"smtp.yandex.ru","Port":465,"User":"u","Password":"p"}`,
		"EMAIL_CONFIG": `{"From":""}`,
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty EMAIL_CONFIG.From")
	}
}

func TestLoadForCLI_WithEnv(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://custom:5432/mydb",
	})

	url, err := LoadForCLI()
	if err != nil {
		t.Fatalf("LoadForCLI() error: %v", err)
	}
	if url != "postgres://custom:5432/mydb" {
		t.Errorf("expected custom URL, got %q", url)
	}
}

func TestLoadForCLI_Default(t *testing.T) {
	os.Unsetenv("DATABASE_URL")

	url, err := LoadForCLI()
	if err != nil {
		t.Fatalf("LoadForCLI() error: %v", err)
	}
	if url == "" {
		t.Error("expected default URL, got empty")
	}
	// Должен быть дефолтный URL
	if url != "postgres://postgres:password@localhost:5432/daily_email_sender?sslmode=disable&client_encoding=UTF-8" {
		t.Errorf("unexpected default URL: %q", url)
	}
}
