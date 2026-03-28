package validation

import (
	"fmt"
	"net/mail"
)

// ValidateEmail проверяет корректность email-адреса.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email не может быть пустым")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("некорректный email '%s'", email)
	}
	return nil
}

// ValidateAge проверяет возраст (13–120 лет).
func ValidateAge(age int) error {
	if age < 13 || age > 120 {
		return fmt.Errorf("возраст должен быть от 13 до 120 лет, получено: %d", age)
	}
	return nil
}

// ValidateHeightCm проверяет рост (100–250 см).
func ValidateHeightCm(height int) error {
	if height < 100 || height > 250 {
		return fmt.Errorf("рост должен быть от 100 до 250 см, получено: %d", height)
	}
	return nil
}

// ValidateWeightKg проверяет вес (30–300 кг).
func ValidateWeightKg(weight float64) error {
	if weight < 30 || weight > 300 {
		return fmt.Errorf("вес должен быть от 30 до 300 кг, получено: %.1f", weight)
	}
	return nil
}

// ValidateDayOfWeek проверяет день недели (0=пн, 6=вс).
func ValidateDayOfWeek(day int) error {
	if day < 0 || day > 6 {
		return fmt.Errorf("день недели должен быть от 0 до 6, получено: %d", day)
	}
	return nil
}

// ValidateHour проверяет час (0–23).
func ValidateHour(hour int) error {
	if hour < 0 || hour > 23 {
		return fmt.Errorf("час должен быть от 0 до 23, получено: %d", hour)
	}
	return nil
}

// ValidateMinute проверяет минуту (0–59).
func ValidateMinute(minute int) error {
	if minute < 0 || minute > 59 {
		return fmt.Errorf("минута должна быть от 0 до 59, получено: %d", minute)
	}
	return nil
}
