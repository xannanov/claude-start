package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"daily-email-sender/internal/database"
	"daily-email-sender/internal/email"
	"daily-email-sender/internal/models"
)

// defaultRetryDelays — задержки между retry-попытками для сетевых ошибок.
var defaultRetryDelays = []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second}

// Generator отвечает за генерацию персонализированного контента через AI.
type Generator struct {
	client         Client
	store          *database.Store
	model          string
	retryDelays    []time.Duration // переопределяется в тестах
	rateLimitDelay time.Duration   // пауза при 429 (rate limit)
}

// NewGenerator создаёт генератор AI-контента.
func NewGenerator(client Client, store *database.Store, model string) *Generator {
	return &Generator{
		client:         client,
		store:          store,
		model:          model,
		retryDelays:    defaultRetryDelays,
		rateLimitDelay: 65 * time.Second,
	}
}

// isRateLimitError проверяет, является ли ошибка rate limit (HTTP 429).
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "Rate limit") || strings.Contains(msg, "rate_limit")
}

// GeneratePersonalizedMessage генерирует персонализированное сообщение через AI.
// При провале всех попыток возвращает fallback-сообщение (не ошибку).
func (g *Generator) GeneratePersonalizedMessage(user models.User, dayOfWeek int, emailType string) models.PersonalizedMessage {
	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	dayName := days[dayOfWeek%7]

	// Получить историю тренировок
	var history []models.WorkoutHistory
	if g.store != nil {
		var err error
		history, err = g.store.GetRecentWorkoutHistory(user.ID, 7)
		if err != nil {
			slog.Warn("не удалось получить историю тренировок", "user_id", user.ID, "error", err)
		}
	}

	ctx := context.Background()

	// Один запрос — тренировка + питание + мотивация
	combined, err := callAndParse(g, ctx, buildCombinedPrompt(user, dayName, emailType, history), 0.85, parseCombinedResponse)
	if err != nil {
		slog.Error("AI: генерация провалилась", "user_id", user.ID, "error", err)
		return g.generateFallbackMessage(user, dayOfWeek, emailType)
	}

	// Сохранить мышечную группу в историю
	if g.store != nil && combined.Workout.MuscleGroup != "" {
		if err := g.store.SaveWorkoutHistory(user.ID, combined.Workout.MuscleGroup); err != nil {
			slog.Warn("не удалось сохранить историю тренировки", "user_id", user.ID, "error", err)
		}
	}

	// Формируем приветствие
	var greeting string
	switch emailType {
	case "morning":
		greeting = "Доброе утро! 🌅"
	case "afternoon":
		greeting = "Добрый день! ☀️"
	default:
		greeting = "Добрый вечер! 🌙"
	}

	return models.PersonalizedMessage{
		Subject:    greeting,
		Motivation: combined.Motivation.Text,
		Workout:    convertWorkout(&combined.Workout),
		Nutrition:  convertNutrition(&combined.Nutrition),
		User:       user,
		DayOfWeek:  dayName,
		TimeOfDay:  emailType,
	}
}

// callAndParse вызывает AI API с retry и парсит ответ. Retry срабатывает и при ошибке API, и при ошибке парсинга.
func callAndParse[T any](g *Generator, ctx context.Context, messages []ChatMessage, temperature float64, parseFn func(string) (*T, error)) (*T, error) {
	var lastErr error

	for attempt, delay := range g.retryDelays {
		resp, err := g.client.ChatCompletion(ctx, ChatRequest{
			Model:       g.model,
			Messages:    messages,
			Temperature: temperature,
			ResponseFormat: &ResponseFormat{
				Type: "json_object",
			},
		})
		if err != nil {
			lastErr = err
			if isRateLimitError(err) {
				slog.Warn("AI: rate limit (429), ожидание перед retry", "attempt", attempt+1, "wait", g.rateLimitDelay)
				time.Sleep(g.rateLimitDelay)
			} else {
				slog.Warn("AI запрос не удался", "attempt", attempt+1, "error", err)
				if attempt < len(g.retryDelays)-1 {
					time.Sleep(delay)
				}
			}
			continue
		}

		content := resp.Choices[0].Message.Content
		if content == "" {
			lastErr = fmt.Errorf("пустой контент в ответе AI")
			slog.Warn("AI вернул пустой контент", "attempt", attempt+1)
			if attempt < len(g.retryDelays)-1 {
				time.Sleep(delay)
			}
			continue
		}

		result, err := parseFn(content)
		if err != nil {
			lastErr = fmt.Errorf("ошибка парсинга: %w", err)
			slog.Warn("AI: ошибка парсинга ответа", "attempt", attempt+1, "error", err)
			if attempt < len(g.retryDelays)-1 {
				time.Sleep(delay)
			}
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("все %d попыток AI исчерпаны: %w", len(g.retryDelays), lastErr)
}

// generateFallbackMessage генерирует сообщение-заглушку с захардкоженными шаблонами.
func (g *Generator) generateFallbackMessage(user models.User, dayOfWeek int, emailType string) models.PersonalizedMessage {
	msg := email.GeneratePersonalizedMessage(user, dayOfWeek, emailType)
	msg.IsFallback = true
	msg.Motivation = fmt.Sprintf(
		"%s, сегодня AI-тренер взял выходной 😅 Но мы подготовили для тебя проверенную тренировку! Завтра вернёмся с индивидуальной программой 💪",
		user.FirstName,
	)
	return msg
}

// convertWorkout конвертирует ответ AI в модель WorkoutPlan.
func convertWorkout(resp *aiWorkoutResponse) models.WorkoutPlan {
	plan := models.WorkoutPlan{
		Title:       resp.Title,
		Duration:    resp.Duration,
		Description: resp.Description,
	}
	for _, ex := range resp.Exercises {
		plan.Exercises = append(plan.Exercises, ex.Name)
		plan.Sets = append(plan.Sets, ex.Sets)
		plan.Reps = append(plan.Reps, ex.Reps)
	}
	return plan
}

// convertNutrition конвертирует ответ AI в модель NutritionPlan.
func convertNutrition(resp *aiNutritionResponse) models.NutritionPlan {
	return models.NutritionPlan{
		Breakfast:     resp.Breakfast,
		Lunch:         resp.Lunch,
		Dinner:        resp.Dinner,
		Snacks:        resp.Snacks,
		CalorieTarget: resp.Calories,
		ProteinGoal:   resp.Protein,
		Fat:           resp.Fat,
		Carbs:         resp.Carbs,
		WaterIntake:   resp.WaterMl,
	}
}
