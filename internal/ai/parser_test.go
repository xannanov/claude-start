package ai

import (
	"testing"
)

func TestExtractJSON_PureJSON(t *testing.T) {
	input := `{"title": "Тренировка", "duration": "30 мин"}`
	got, err := extractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestExtractJSON_MarkdownJSONBlock(t *testing.T) {
	input := "Вот ваша тренировка:\n```json\n{\"title\": \"Силовая\"}\n```\nУспехов!"
	got, err := extractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"title": "Силовая"}` {
		t.Errorf("got %q", got)
	}
}

func TestExtractJSON_MarkdownBlock(t *testing.T) {
	input := "```\n{\"text\": \"Привет!\"}\n```"
	got, err := extractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"text": "Привет!"}` {
		t.Errorf("got %q", got)
	}
}

func TestExtractJSON_JSONWithSurroundingText(t *testing.T) {
	input := `Вот план: {"breakfast": "Каша", "lunch": "Суп"} - готово!`
	got, err := extractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"breakfast": "Каша", "lunch": "Суп"}` {
		t.Errorf("got %q", got)
	}
}

func TestExtractJSON_InvalidJSON(t *testing.T) {
	input := "это просто текст без JSON"
	_, err := extractJSON(input)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseCombinedResponse_Valid(t *testing.T) {
	raw := `{
		"workout": {
			"title": "Силовая тренировка на грудь",
			"muscle_group": "chest",
			"duration": "45 минут",
			"description": "Тренировка для набора мышечной массы",
			"exercises": [
				{"name": "Жим лёжа", "sets": "4 подхода", "reps": "8-10 раз"},
				{"name": "Разводка гантелей", "sets": "3 подхода", "reps": "12 раз"}
			]
		},
		"nutrition": {
			"breakfast": "Овсянка 150г + яйцо 2шт",
			"lunch": "Куриная грудка 200г + рис 150г",
			"dinner": "Лосось 180г + овощи",
			"snacks": ["Творог 200г", "Банан"],
			"calories": "2800 ккал",
			"protein": "180 г",
			"fat": "80 г",
			"carbs": "300 г",
			"water_ml": "2800 мл"
		},
		"motivation": {
			"text": "Алексей, сегодня твой день! 💪 Грудь не прокачает себя сама!"
		}
	}`
	resp, err := parseCombinedResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Workout.Title != "Силовая тренировка на грудь" {
		t.Errorf("workout.title = %q", resp.Workout.Title)
	}
	if resp.Workout.MuscleGroup != "chest" {
		t.Errorf("workout.muscle_group = %q", resp.Workout.MuscleGroup)
	}
	if len(resp.Workout.Exercises) != 2 {
		t.Errorf("exercises count = %d, want 2", len(resp.Workout.Exercises))
	}
	if resp.Nutrition.Breakfast != "Овсянка 150г + яйцо 2шт" {
		t.Errorf("nutrition.breakfast = %q", resp.Nutrition.Breakfast)
	}
	if resp.Nutrition.Calories != "2800 ккал" {
		t.Errorf("nutrition.calories = %q", resp.Nutrition.Calories)
	}
	if resp.Motivation.Text == "" {
		t.Error("motivation.text is empty")
	}
}

func TestParseCombinedResponse_MissingWorkoutTitle(t *testing.T) {
	raw := `{
		"workout": {"muscle_group": "chest", "duration": "30 мин", "exercises": [{"name": "Жим", "sets": "3", "reps": "10"}]},
		"nutrition": {"breakfast": "Каша", "lunch": "Суп", "dinner": "Рыба", "calories": "2000 ккал"},
		"motivation": {"text": "Давай!"}
	}`
	_, err := parseCombinedResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing workout.title")
	}
}

func TestParseCombinedResponse_NoExercises(t *testing.T) {
	raw := `{
		"workout": {"title": "Тренировка", "duration": "30 мин", "exercises": []},
		"nutrition": {"breakfast": "Каша", "lunch": "Суп", "dinner": "Рыба", "calories": "2000 ккал"},
		"motivation": {"text": "Давай!"}
	}`
	_, err := parseCombinedResponse(raw)
	if err == nil {
		t.Fatal("expected error for empty exercises")
	}
}

func TestParseCombinedResponse_MissingCalories(t *testing.T) {
	raw := `{
		"workout": {"title": "Тренировка", "duration": "30 мин", "exercises": [{"name": "Бег", "sets": "1", "reps": "20 мин"}]},
		"nutrition": {"breakfast": "Каша", "lunch": "Суп", "dinner": "Рыба"},
		"motivation": {"text": "Давай!"}
	}`
	_, err := parseCombinedResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing nutrition.calories")
	}
}

func TestParseCombinedResponse_EmptyMotivation(t *testing.T) {
	raw := `{
		"workout": {"title": "Тренировка", "duration": "30 мин", "exercises": [{"name": "Бег", "sets": "1", "reps": "20 мин"}]},
		"nutrition": {"breakfast": "Каша", "lunch": "Суп", "dinner": "Рыба", "calories": "2000 ккал"},
		"motivation": {"text": ""}
	}`
	_, err := parseCombinedResponse(raw)
	if err == nil {
		t.Fatal("expected error for empty motivation.text")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q", got)
	}
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("got %q", got)
	}
}
