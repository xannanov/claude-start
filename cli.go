package main

import (
	"bufio"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
)

// CLI provides command-line interface functions
type CLI struct{}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{}
}

// AddUserInteractive creates a user interactively through CLI prompts
func (cli *CLI) AddUserInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)

	// Connect to database
	if err := ConnectToDatabase(); err != nil {
		return fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}
	defer CloseDatabase()

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
		if _, parseErr := mail.ParseAddress(email); parseErr != nil {
			fmt.Printf("Некорректный email '%s'. Попробуйте снова: ", email)
			continue
		}
		break
	}

	// First name
	fmt.Print("Имя (First Name): ")
	firstName := cli.readString(scanner)

	// Last name
	fmt.Print("Фамилия (Last Name): ")
	lastName := cli.readString(scanner)

	// Age
	var age int
	for {
		fmt.Print("Возраст (13–120): ")
		var err error
		age, err = cli.readInt(scanner)
		if err != nil {
			fmt.Printf("Неверный возраст: %v. Попробуйте снова.\n", err)
			continue
		}
		if age < 13 || age > 120 {
			fmt.Printf("Возраст должен быть от 13 до 120. Введено: %d\n", age)
			continue
		}
		break
	}

	// Gender
	fmt.Print("Пол (male/female/other): ")
	gender := strings.ToLower(cli.readString(scanner))
	for gender != "male" && gender != "female" && gender != "other" {
		fmt.Print("Пол должен быть male, female или other. Попробуйте снова: ")
		gender = strings.ToLower(cli.readString(scanner))
	}

	// Height in cm
	var height int
	for {
		fmt.Print("Рост (100–250 см): ")
		var err error
		height, err = cli.readInt(scanner)
		if err != nil {
			fmt.Printf("Неверный рост: %v. Попробуйте снова.\n", err)
			continue
		}
		if height < 100 || height > 250 {
			fmt.Printf("Рост должен быть от 100 до 250 см. Введено: %d\n", height)
			continue
		}
		break
	}

	// Weight in kg
	var weight float64
	for {
		fmt.Print("Вес (30–300 кг): ")
		var err error
		weight, err = cli.readFloat(scanner)
		if err != nil {
			fmt.Printf("Неверный вес: %v. Попробуйте снова.\n", err)
			continue
		}
		if weight < 30 || weight > 300 {
			fmt.Printf("Вес должен быть от 30 до 300 кг. Введено: %.1f\n", weight)
			continue
		}
		break
	}

	// Goal
	fmt.Print("Цель (weight_loss/muscle_gain/maintenance/general_fitness): ")
	goal := strings.ToLower(cli.readString(scanner))
	validGoals := []string{"weight_loss", "muscle_gain", "maintenance", "general_fitness"}
	for !contains(validGoals, goal) {
		fmt.Printf("Цель должна быть одной из: %v. Попробуйте снова: ", validGoals)
		goal = strings.ToLower(cli.readString(scanner))
	}

	// Activity level
	fmt.Print("Уровень активности (sedentary/light/moderate/active/very_active): ")
	activityLevel := strings.ToLower(cli.readString(scanner))
	validLevels := []string{"sedentary", "light", "moderate", "active", "very_active"}
	for !contains(validLevels, activityLevel) {
		fmt.Printf("Уровень активности должен быть одной из: %v. Попробуйте снова: ", validLevels)
		activityLevel = strings.ToLower(cli.readString(scanner))
	}

	// Check for duplicate email
	if _, err := GetUserByEmail(email); err == nil {
		return fmt.Errorf("пользователь с email '%s' уже существует", email)
	}

	// Create user
	user := &User{
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

	if err := CreateUser(db, user); err != nil {
		return fmt.Errorf("ошибка при создании пользователя: %w", err)
	}

	// Add default schedule (every Monday at 8:00 AM)
	defaultSchedule := &UserSchedule{
		UserID:    user.ID,
		DayOfWeek: 0, // Monday
		TimeHour:  8,
		TimeMinute: 0,
		EmailType: "morning",
	}

	if err := CreateUserSchedule(db, defaultSchedule); err != nil {
		return fmt.Errorf("ошибка при создании расписания: %w", err)
	}

	fmt.Printf("\n✓ Пользователь успешно создан!\n")
	fmt.Printf("  ID: %s\n", user.ID)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Имя: %s %s\n", user.FirstName, user.LastName)
	fmt.Printf("  Цель: %s\n", user.Goal)
	fmt.Printf("  Расписание: %s, %d:%02d (утро)\n", getDayName(0), defaultSchedule.TimeHour, defaultSchedule.TimeMinute)

	return nil
}

// ListUsers lists all users in the database
func (cli *CLI) ListUsers() error {
	// Ensure database is connected
	if err := ConnectToDatabase(); err != nil {
		return fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}
	defer CloseDatabase()

	users, err := GetAllUsers()
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

	for _, user := range users {
		fmt.Printf("%-40s %-20s %-15s %-10.2f кг\n",
			user.Email,
			fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			user.Goal,
			user.WeightKg,
		)
	}

	fmt.Println()
	return nil
}

// readString reads a string from stdin, ignoring empty lines
func (cli *CLI) readString(scanner *bufio.Scanner) string {
	for scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// readInt reads an integer from stdin
func (cli *CLI) readInt(scanner *bufio.Scanner) (int, error) {
	text := cli.readString(scanner)
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("неверное число: %s", text)
	}
	return value, nil
}

// readFloat reads a float64 from stdin
func (cli *CLI) readFloat(scanner *bufio.Scanner) (float64, error) {
	text := cli.readString(scanner)
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0.0, fmt.Errorf("неверное число: %s", text)
	}
	return value, nil
}

// AddScheduleInteractive adds a schedule interactively for an existing user
func (cli *CLI) AddScheduleInteractive() error {
	fmt.Println("\n=== Добавление расписания для пользователя ===")

	// Ensure database is connected
	if err := ConnectToDatabase(); err != nil {
		return fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}
	defer CloseDatabase()

	users, err := GetAllUsers()
	if err != nil {
		return fmt.Errorf("ошибка при получении списка пользователей: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("нет пользователей в базе данных")
	}

	// Display users
	fmt.Println("Доступные пользователи:")
	for i, user := range users {
		fmt.Printf("%d. %s (%s)\n", i+1, user.Email, user.ID)
	}

	// Select user
	fmt.Printf("\nВыберите пользователя (1-%d): ", len(users))
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		choice, err := strconv.Atoi(scanner.Text())
		if err != nil || choice < 1 || choice > len(users) {
			fmt.Printf("Пожалуйста, введите число от 1 до %d: ", len(users))
			continue
		}
		selectedUser := users[choice-1]

		// Add schedule
		fmt.Printf("\nДобавление расписания для: %s\n", selectedUser.Email)

		fmt.Print("День недели (0-Пн, 1-Вт, 2-Ср, 3-Чт, 4-Пт, 5-Сб, 6-Вс): ")
		day, err := cli.readInt(scanner)
		if err != nil || day < 0 || day > 6 {
			return fmt.Errorf("неверный день: ожидается 0–6")
		}

		fmt.Print("Час (0-23): ")
		hour, err := cli.readInt(scanner)
		if err != nil || hour < 0 || hour > 23 {
			return fmt.Errorf("неверный час: ожидается 0–23")
		}

		fmt.Print("Минута (0-59): ")
		minute, err := cli.readInt(scanner)
		if err != nil || minute < 0 || minute > 59 {
			return fmt.Errorf("неверная минута: ожидается 0–59")
		}

		fmt.Print("Тип email (morning/afternoon/evening): ")
		emailType := cli.readString(scanner)
		emailType = strings.ToLower(emailType)
		validTypes := []string{"morning", "afternoon", "evening"}
		for !contains(validTypes, emailType) {
			fmt.Printf("Тип должен быть одной из: %v. Попробуйте снова: ", validTypes)
			emailType = strings.ToLower(cli.readString(scanner))
		}

		schedule := &UserSchedule{
			UserID:      selectedUser.ID,
			DayOfWeek:   day,
			TimeHour:    hour,
			TimeMinute:  minute,
			EmailType:   emailType,
			IsActive:    true,
		}

		if err := CreateUserSchedule(db, schedule); err != nil {
			return fmt.Errorf("ошибка при создании расписания: %w", err)
		}

		fmt.Printf("\n✓ Расписание успешно добавлено!\n")
		fmt.Printf("  Email: %s\n", selectedUser.Email)
		fmt.Printf("  День: %s, Время: %d:%02d, Тип: %s\n",
			getDayName(day), hour, minute, emailType)

		return nil
	}

	return nil
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
