package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
)

type SMTPConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
}

type EmailConfig struct {
	From string `json:"From"`
	To   string `json:"To"`
}

type Message struct {
	Subject string
	Body    string
	Time    string
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Parse SMTP config from env
	smtpConfigStr := getEnv("SMTP_CONFIG", `{"Host":"smtp.yandex.ru","Port":465,"User":"your-email@yandex.com","Password":"your-password"}`)
	var smtpConfig SMTPConfig
	if err := json.Unmarshal([]byte(smtpConfigStr), &smtpConfig); err != nil {
		log.Printf("Error parsing SMTP config: %v", err)
		os.Exit(1)
	}

	// Parse email config from env
	emailConfigStr := getEnv("EMAIL_CONFIG", `{"From":"sender@example.com","To":"recipient@example.com"}`)
	var emailConfig EmailConfig
	if err := json.Unmarshal([]byte(emailConfigStr), &emailConfig); err != nil {
		log.Printf("Error parsing email config: %v", err)
		os.Exit(1)
	}

	if smtpConfig.Password == "your-password" || smtpConfig.User == "your-email@yandex.com" {
		log.Println("Please configure SMTP_CONFIG and EMAIL_CONFIG in .env file")
		os.Exit(1)
	}

	// Get the current hour (0-23)
	now := time.Now()
	hour := now.Hour()

	var message Message

	if hour < 12 {
		// Morning (before 12:00)
		message = Message{
			Subject: "Доброе утро! 🌅",
			Body:    generateMorningMessage(now),
			Time:    "morning",
		}
	} else if hour < 18 {
		// Afternoon (12:00 - 18:00)
		message = Message{
			Subject: "Добрый день! ☀️",
			Body:    generateAfternoonMessage(now),
			Time:    "afternoon",
		}
	} else {
		// Evening (after 18:00)
		message = Message{
			Subject: "Добрый вечер! 🌙",
			Body:    generateEveningMessage(now),
			Time:    "evening",
		}
	}

	sendEmail(smtpConfig, emailConfig, message)
}

func sendEmail(smtp SMTPConfig, email EmailConfig, message Message) {
	m := gomail.NewMessage()
	m.SetHeader("From", email.From)
	m.SetHeader("To", email.To)
	m.SetHeader("Subject", message.Subject)
	m.SetBody("text/html", message.Body)

	d := gomail.NewDialer(smtp.Host, smtp.Port, smtp.User, smtp.Password)
	d.SSL = true
	d.TLSConfig = &tls.Config{ServerName: smtp.Host}

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email: %v", err)
		os.Exit(1)
	}

	log.Printf("Email sent successfully at %s: %s", message.Time, message.Subject)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateMorningMessage(now time.Time) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #f0f0f0; padding: 20px;">
			<div style="background-color: #fff; border-radius: 10px; padding: 30px; max-width: 600px; margin: 0 auto; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
				<h1 style="color: #4CAF50;">Доброе утро! 🌅</h1>
				<p>Приветствую тебя в этот прекрасный день!</p>
				<p>Сегодня %s</p>
				<div style="background-color: #e8f5e9; padding: 15px; border-radius: 5px; margin: 20px 0;">
					<h3 style="margin-top: 0;">Утренние мысли:</h3>
					<ul>
						<li>Начни день с улыбки</li>
						<li>Поставь приоритеты на сегодня</li>
						<li>Будь в гармонии с собой</li>
					</ul>
				</div>
				<p>Желаю продуктивного и счастливого дня!</p>
				<p style="color: #888; font-size: 12px;">Время: %s</p>
			</div>
		</body>
		</html>
	`, now.Format("2 January 2006"), now.Format("15:04:05"))
}

func generateAfternoonMessage(now time.Time) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #fff5e6; padding: 20px;">
			<div style="background-color: #fff; border-radius: 10px; padding: 30px; max-width: 600px; margin: 0 auto; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
				<h1 style="color: #FF9800;">Добрый день! ☀️</h1>
				<p>Приветствую тебя на середине дня!</p>
				<p>Сегодня %s</p>
				<div style="background-color: #fff3e0; padding: 15px; border-radius: 5px; margin: 20px 0;">
					<h3 style="margin-top: 0;">Моментальный отдых:</h3>
					<ul>
						<li>Вдохни и выдохни</li>
						<li>Выпей стакан воды</li>
						<li>Отдохни пару минут</li>
					</ul>
				</div>
				<p>Ты делаешь замечательные вещи!</p>
				<p style="color: #888; font-size: 12px;">Время: %s</p>
			</div>
		</body>
		</html>
	`, now.Format("2 January 2006"), now.Format("15:04:05"))
}

func generateEveningMessage(now time.Time) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #f3e5f5; padding: 20px;">
			<div style="background-color: #fff; border-radius: 10px; padding: 30px; max-width: 600px; margin: 0 auto; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
				<h1 style="color: #9C27B0;">Добрый вечер! 🌙</h1>
				<p>Приветствую тебя в конце дня!</p>
				<p>Сегодня %s</p>
				<div style="background-color: #f3e5f5; padding: 15px; border-radius: 5px; margin: 20px 0;">
					<h3 style="margin-top: 0;">Вечерние размышления:</h3>
					<ul>
						<li>Что ты сегодня сделал хорошо?</li>
						<li>Что можно улучшить завтра?</li>
						<li>Отдохни и восстанови силы</li>
					</ul>
				</div>
				<p>Спасибо тебе за сегодняшний день!</p>
				<p style="color: #888; font-size: 12px;">Время: %s</p>
			</div>
		</body>
		</html>
	`, now.Format("2 January 2006"), now.Format("15:04:05"))
}
