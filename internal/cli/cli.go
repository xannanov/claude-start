package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"daily-email-sender/internal/database"
	"daily-email-sender/internal/models"
	"daily-email-sender/internal/validation"
)

// CLI предоставляет интерактивный интерфейс командной строки.
type CLI struct {
	store *database.Store
}

// New создаёт CLI с подключением к БД.
func New(store *database.Store) *CLI {
	return &CLI{store: store}
}

// AddUserInteractive создаёт пользователя через интерактивные подсказки.
func (c *CLI) AddUserInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\n=== Добавление нового пользователя ===")

	// Email
	fmt.Print("Email: ")
	var email string
	for scanner.Scan() {
		email = strings.TrimSpace(scanner.Text())
		if email == "" {
			fmt.Print("Email (не может быть пустым): ")
			continue
		}
		if err := validation.ValidateEmail(email); err != nil {
			fmt.Printf("%s. Попробуйте снова: ", err)
			continue
		}
		break
	}

	// Проверка дубликата email
	if _, err := c.store.GetUserByEmail(email); err == nil {
		return fmt.Errorf("пользователь с email '%s' уже существует", email)
	}

	// Имя
	fmt.Print("Имя: ")
	firstName := readString(scanner)

	// Фамилия
	fmt.Print("Фамилия: ")
	lastName := readString(scanner)

	// Возраст
	age := readIntInRange(scanner, "Возраст (13–120): ", 13, 120)

	// Пол
	fmt.Print("Пол (male/female/other): ")
	gender := strings.ToLower(readString(scanner))
	for gender != "male" && gender != "female" && gender != "other" {
		fmt.Print("Пол должен быть male, female или other. Попробуйте снова: ")
		gender = strings.ToLower(readString(scanner))
	}

	// Рост
	height := readIntInRange(scanner, "Рост (100–250 см): ", 100, 250)

	// Вес
	weight := readFloatInRange(scanner, "Вес (30–300 кг): ", 30, 300)

	// Цель
	validGoals := []string{"weight_loss", "muscle_gain", "maintenance", "general_fitness"}
	fmt.Print("Цель (weight_loss/muscle_gain/maintenance/general_fitness): ")
	goal := strings.ToLower(readString(scanner))
	for !contains(validGoals, goal) {
		fmt.Printf("Цель должна быть одной из: %v. Попробуйте снова: ", validGoals)
		goal = strings.ToLower(readString(scanner))
	}

	// Уровень активности
	validLevels := []string{"sedentary", "light", "moderate", "active", "very_active"}
	fmt.Print("Уровень активности (sedentary/light/moderate/active/very_active): ")
	activityLevel := strings.ToLower(readString(scanner))
	for !contains(validLevels, activityLevel) {
		fmt.Printf("Уровень активности должен быть одним из: %v. Попробуйте снова: ", validLevels)
		activityLevel = strings.ToLower(readString(scanner))
	}

	user := &models.User{
		Email:         email,
		FirstName:     strings.TrimSpace(firstName),
		LastName:      strings.TrimSpace(lastName),
		Age:           age,
		Gender:        gender,
		HeightCm:      height,
		WeightKg:      weight,
		Goal:          goal,
		ActivityLevel: activityLevel,
	}

	if err := c.store.CreateUser(user); err != nil {
		return fmt.Errorf("ошибка при создании пользователя: %w", err)
	}

	// Расписание по умолчанию: каждый понедельник в 8:00
	defaultSchedule := &models.UserSchedule{
		UserID:     user.ID,
		DayOfWeek:  0, // Понедельник
		TimeHour:   8,
		TimeMinute: 0,
		EmailType:  "morning",
	}
	if err := c.store.CreateUserSchedule(defaultSchedule); err != nil {
		return fmt.Errorf("ошибка при создании расписания: %w", err)
	}

	fmt.Printf("\n✓ Пользователь успешно создан!\n")
	fmt.Printf("  ID: %s\n", user.ID)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Имя: %s %s\n", user.FirstName, user.LastName)
	fmt.Printf("  Цель: %s\n", user.Goal)
	fmt.Printf("  Расписание: %s, %d:%02d (утро)\n",
		getDayName(defaultSchedule.DayOfWeek), defaultSchedule.TimeHour, defaultSchedule.TimeMinute)

	return nil
}

// ListUsers выводит всех пользователей в виде таблицы.
func (c *CLI) ListUsers() error {
	users, err := c.store.GetAllUsers()
	if err != nil {
		return fmt.Errorf("ошибка при получении списка пользователей: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("Пользователей пока нет.")
		return nil
	}

	fmt.Printf("\n=== Список пользователей (%d) ===\n", len(users))
	fmt.Printf("%-40s %-20s %-15s %-10s\n", "Email", "Имя", "Цель", "Вес")
	fmt.Println(strings.Repeat("-", 85))
	for _, u := range users {
		fmt.Printf("%-40s %-20s %-15s %-10.2f кг\n",
			u.Email,
			fmt.Sprintf("%s %s", u.FirstName, u.LastName),
			u.Goal,
			u.WeightKg,
		)
	}
	fmt.Println()
	return nil
}

// AddScheduleInteractive добавляет расписание для существующего пользователя.
func (c *CLI) AddScheduleInteractive() error {
	fmt.Println("\n=== Добавление расписания для пользователя ===")

	users, err := c.store.GetAllUsers()
	if err != nil {
		return fmt.Errorf("ошибка при получении списка пользователей: %w", err)
	}
	if len(users) == 0 {
		return fmt.Errorf("нет пользователей в базе данных")
	}

	fmt.Println("Доступные пользователи:")
	for i, u := range users {
		fmt.Printf("%d. %s (%s)\n", i+1, u.Email, u.ID)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("\nВыберите пользователя (1-%d): ", len(users))

	var selectedUser models.User
	for scanner.Scan() {
		choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil || choice < 1 || choice > len(users) {
			fmt.Printf("Пожалуйста, введите число от 1 до %d: ", len(users))
			continue
		}
		selectedUser = users[choice-1]
		break
	}

	fmt.Printf("\nДобавление расписания для: %s\n", selectedUser.Email)

	fmt.Print("День недели (0-Пн, 1-Вт, 2-Ср, 3-Чт, 4-Пт, 5-Сб, 6-Вс): ")
	day, err := readIntValidated(scanner)
	if err != nil || day < 0 || day > 6 {
		return fmt.Errorf("неверный день: ожидается 0–6")
	}

	fmt.Print("Час (0-23): ")
	hour, err := readIntValidated(scanner)
	if err != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("неверный час: ожидается 0–23")
	}

	fmt.Print("Минута (0-59): ")
	minute, err := readIntValidated(scanner)
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("неверная минута: ожидается 0–59")
	}

	validTypes := []string{"morning", "afternoon", "evening"}
	fmt.Print("Тип email (morning/afternoon/evening): ")
	emailType := strings.ToLower(readString(scanner))
	for !contains(validTypes, emailType) {
		fmt.Printf("Тип должен быть одним из: %v. Попробуйте снова: ", validTypes)
		emailType = strings.ToLower(readString(scanner))
	}

	schedule := &models.UserSchedule{
		UserID:     selectedUser.ID,
		DayOfWeek:  day,
		TimeHour:   hour,
		TimeMinute: minute,
		EmailType:  emailType,
		IsActive:   true,
	}

	if err := c.store.CreateUserSchedule(schedule); err != nil {
		return fmt.Errorf("ошибка при создании расписания: %w", err)
	}

	fmt.Printf("\n✓ Расписание успешно добавлено!\n")
	fmt.Printf("  Email: %s\n", selectedUser.Email)
	fmt.Printf("  День: %s, Время: %d:%02d, Тип: %s\n",
		getDayName(day), hour, minute, emailType)

	return nil
}

// --- вспомогательные функции ---

func readString(scanner *bufio.Scanner) string {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func readIntValidated(scanner *bufio.Scanner) (int, error) {
	text := readString(scanner)
	return strconv.Atoi(text)
}

func readIntInRange(scanner *bufio.Scanner, prompt string, min, max int) int {
	for {
		fmt.Print(prompt)
		v, err := readIntValidated(scanner)
		if err != nil {
			fmt.Printf("Неверное число. Попробуйте снова.\n")
			continue
		}
		if v < min || v > max {
			fmt.Printf("Значение должно быть от %d до %d. Введено: %d\n", min, max, v)
			continue
		}
		return v
	}
}

func readFloatInRange(scanner *bufio.Scanner, prompt string, min, max float64) float64 {
	for {
		fmt.Print(prompt)
		text := readString(scanner)
		v, err := strconv.ParseFloat(text, 64)
		if err != nil {
			fmt.Printf("Неверное число. Попробуйте снова.\n")
			continue
		}
		if v < min || v > max {
			fmt.Printf("Значение должно быть от %.0f до %.0f. Введено: %.1f\n", min, max, v)
			continue
		}
		return v
	}
}

func getDayName(day int) string {
	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	if day >= 0 && day < len(days) {
		return days[day]
	}
	return "Неизвестный день"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
