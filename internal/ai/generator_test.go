package ai

import (
	"context"
	"fmt"
	"testing"
	"time"

	"daily-email-sender/internal/models"
)

// mockClient — мок AI-клиента для тестов.
type mockClient struct {
	responses []string // ответы по порядку вызовов
	errors    []error
	callIndex int
}

func (m *mockClient) ChatCompletion(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	idx := m.callIndex
	m.callIndex++

	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}

	content := ""
	if idx < len(m.responses) {
		content = m.responses[idx]
	}

	return &ChatResponse{
		Choices: []ChatChoice{
			{Message: ChatMessage{Role: "assistant", Content: content}},
		},
	}, nil
}

// noDelays — нулевые задержки для тестов.
var noDelays = []time.Duration{0, 0, 0}

var testUser = models.User{
	ID:            "test-uuid",
	Email:         "test@example.com",
	FirstName:     "Алексей",
	LastName:      "Тестов",
	Age:           30,
	Gender:        "male",
	HeightCm:      180,
	WeightKg:      80.0,
	Goal:          "muscle_gain",
	ActivityLevel: "active",
}

var validWorkoutJSON = `{
	"title": "Силовая тренировка на грудь",
	"muscle_group": "chest",
	"duration": "45 минут",
	"description": "Тренировка для набора мышечной массы",
	"exercises": [
		{"name": "Жим лёжа", "sets": "4 подхода", "reps": "8-10 раз"},
		{"name": "Жим гантелей на наклонной", "sets": "3 подхода", "reps": "10-12 раз"},
		{"name": "Разводка гантелей", "sets": "3 подхода", "reps": "12 раз"},
		{"name": "Кроссовер", "sets": "3 подхода", "reps": "12-15 раз"}
	]
}`

var validNutritionJSON = `{
	"breakfast": "Яичница из 3 яиц с тостами (200 г)",
	"lunch": "Куриная грудка (200 г) с рисом (150 г) и салатом",
	"dinner": "Лосось (180 г) с овощами на пару (200 г)",
	"snacks": ["Творог 5% (200 г)", "Банан", "Протеиновый коктейль"],
	"calories": "2800 ккал",
	"protein": "180 г",
	"fat": "80 г",
	"carbs": "300 г",
	"water_ml": "2800 мл"
}`

var validMotivationJSON = `{"text": "Алексей, подъём! 💪 Грудь сама себя не прокачает! Давай, чемпион! 🏆"}`

func TestGenerator_SuccessfulGeneration(t *testing.T) {
	mock := &mockClient{
		responses: []string{validWorkoutJSON, validNutritionJSON, validMotivationJSON},
	}

	gen := &Generator{
		client:      mock,
		store:       nil,
		model:       "deepseek-chat",
		retryDelays: noDelays,
	}

	msg := gen.GeneratePersonalizedMessage(testUser, 0, "morning")

	if msg.IsFallback {
		t.Error("expected AI message, got fallback")
	}
	if msg.Workout.Title != "Силовая тренировка на грудь" {
		t.Errorf("workout title = %q", msg.Workout.Title)
	}
	if len(msg.Workout.Exercises) != 4 {
		t.Errorf("exercises count = %d, want 4", len(msg.Workout.Exercises))
	}
	if msg.Nutrition.Breakfast != "Яичница из 3 яиц с тостами (200 г)" {
		t.Errorf("breakfast = %q", msg.Nutrition.Breakfast)
	}
	if msg.Nutrition.Fat != "80 г" {
		t.Errorf("fat = %q", msg.Nutrition.Fat)
	}
	if msg.Nutrition.Carbs != "300 г" {
		t.Errorf("carbs = %q", msg.Nutrition.Carbs)
	}
	if msg.Motivation == "" {
		t.Error("expected non-empty motivation")
	}
	if msg.Subject != "Доброе утро! 🌅" {
		t.Errorf("subject = %q", msg.Subject)
	}
	if msg.DayOfWeek != "Понедельник" {
		t.Errorf("dayOfWeek = %q", msg.DayOfWeek)
	}
}

func TestGenerator_FallbackOnAllWorkoutRetryFailed(t *testing.T) {
	mock := &mockClient{
		errors: []error{
			fmt.Errorf("timeout"),
			fmt.Errorf("timeout"),
			fmt.Errorf("timeout"),
		},
	}

	gen := &Generator{
		client:      mock,
		store:       nil,
		model:       "deepseek-chat",
		retryDelays: noDelays,
	}

	msg := gen.GeneratePersonalizedMessage(testUser, 0, "morning")

	if !msg.IsFallback {
		t.Error("expected fallback message")
	}
	if msg.Motivation == "" {
		t.Error("expected fallback motivation text")
	}
	if msg.Workout.Title == "" {
		t.Error("expected non-empty fallback workout title")
	}
}

func TestGenerator_FallbackOnNutritionParseFailure(t *testing.T) {
	mock := &mockClient{
		responses: []string{
			validWorkoutJSON,  // тренировка OK
			"invalid json",    // питание fail 1
			"invalid json",    // питание fail 2
			"invalid json",    // питание fail 3
		},
	}

	gen := &Generator{
		client:      mock,
		store:       nil,
		model:       "deepseek-chat",
		retryDelays: noDelays,
	}

	msg := gen.GeneratePersonalizedMessage(testUser, 2, "afternoon")

	if !msg.IsFallback {
		t.Error("expected fallback on nutrition parse failure")
	}
}

func TestGenerator_RetrySucceedsOnSecondAttempt(t *testing.T) {
	mock := &mockClient{
		responses: []string{
			"",                    // тренировка fail (пустой контент)
			validWorkoutJSON,      // тренировка OK (retry)
			validNutritionJSON,    // питание OK
			validMotivationJSON,   // мотивация OK
		},
		errors: []error{nil, nil, nil, nil},
	}

	gen := &Generator{
		client:      mock,
		store:       nil,
		model:       "deepseek-chat",
		retryDelays: noDelays,
	}

	msg := gen.GeneratePersonalizedMessage(testUser, 4, "evening")

	if msg.IsFallback {
		t.Error("expected AI message after retry, got fallback")
	}
	if msg.Subject != "Добрый вечер! 🌙" {
		t.Errorf("subject = %q", msg.Subject)
	}
}

func TestGenerator_GreetingByTimeOfDay(t *testing.T) {
	tests := []struct {
		emailType string
		want      string
	}{
		{"morning", "Доброе утро! 🌅"},
		{"afternoon", "Добрый день! ☀️"},
		{"evening", "Добрый вечер! 🌙"},
	}

	for _, tt := range tests {
		t.Run(tt.emailType, func(t *testing.T) {
			mock := &mockClient{
				responses: []string{validWorkoutJSON, validNutritionJSON, validMotivationJSON},
			}
			gen := &Generator{client: mock, store: nil, model: "deepseek-chat", retryDelays: noDelays}
			msg := gen.GeneratePersonalizedMessage(testUser, 0, tt.emailType)
			if msg.Subject != tt.want {
				t.Errorf("subject = %q, want %q", msg.Subject, tt.want)
			}
		})
	}
}

func TestConvertWorkout(t *testing.T) {
	resp := &aiWorkoutResponse{
		Title:       "Тренировка",
		Duration:    "30 мин",
		Description: "Описание",
		Exercises: []aiExercise{
			{Name: "Жим", Sets: "3 подхода", Reps: "10 раз"},
			{Name: "Тяга", Sets: "4 подхода", Reps: "8 раз"},
		},
	}
	plan := convertWorkout(resp)

	if plan.Title != "Тренировка" {
		t.Errorf("title = %q", plan.Title)
	}
	if len(plan.Exercises) != 2 {
		t.Fatalf("exercises = %d", len(plan.Exercises))
	}
	if plan.Exercises[0] != "Жим" {
		t.Errorf("exercise[0] = %q", plan.Exercises[0])
	}
	if plan.Sets[1] != "4 подхода" {
		t.Errorf("sets[1] = %q", plan.Sets[1])
	}
}

func TestConvertNutrition(t *testing.T) {
	resp := &aiNutritionResponse{
		Breakfast: "Каша",
		Lunch:     "Суп",
		Dinner:    "Рыба",
		Snacks:    []string{"Яблоко"},
		Calories:  "2000 ккал",
		Protein:   "100 г",
		Fat:       "65 г",
		Carbs:     "250 г",
		WaterMl:   "2500 мл",
	}
	plan := convertNutrition(resp)

	if plan.CalorieTarget != "2000 ккал" {
		t.Errorf("calories = %q", plan.CalorieTarget)
	}
	if plan.Fat != "65 г" {
		t.Errorf("fat = %q", plan.Fat)
	}
	if plan.WaterIntake != "2500 мл" {
		t.Errorf("water = %q", plan.WaterIntake)
	}
}
