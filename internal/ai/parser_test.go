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

func TestParseWorkoutResponse_Valid(t *testing.T) {
	raw := `{
		"title": "Силовая тренировка",
		"muscle_group": "chest",
		"duration": "45 минут",
		"description": "Тренировка на грудь",
		"exercises": [
			{"name": "Жим лёжа", "sets": "4 подхода", "reps": "8-10 раз"},
			{"name": "Разводка", "sets": "3 подхода", "reps": "12 раз"}
		]
	}`
	resp, err := parseWorkoutResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Title != "Силовая тренировка" {
		t.Errorf("title = %q", resp.Title)
	}
	if resp.MuscleGroup != "chest" {
		t.Errorf("muscle_group = %q", resp.MuscleGroup)
	}
	if len(resp.Exercises) != 2 {
		t.Errorf("exercises count = %d, want 2", len(resp.Exercises))
	}
}

func TestParseWorkoutResponse_MissingTitle(t *testing.T) {
	raw := `{"duration": "30 мин", "exercises": [{"name": "Бег", "sets": "1", "reps": "30 мин"}]}`
	_, err := parseWorkoutResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseWorkoutResponse_NoExercises(t *testing.T) {
	raw := `{"title": "Тренировка", "duration": "30 мин", "exercises": []}`
	_, err := parseWorkoutResponse(raw)
	if err == nil {
		t.Fatal("expected error for empty exercises")
	}
}

func TestParseNutritionResponse_Valid(t *testing.T) {
	raw := `{
		"breakfast": "Каша с ягодами (200 г)",
		"lunch": "Куриная грудка с рисом (300 г)",
		"dinner": "Рыба с овощами (250 г)",
		"snacks": ["Яблоко", "Орехи (30 г)"],
		"calories": "2000 ккал",
		"protein": "120 г",
		"fat": "65 г",
		"carbs": "250 г",
		"water_ml": "2500 мл"
	}`
	resp, err := parseNutritionResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Breakfast != "Каша с ягодами (200 г)" {
		t.Errorf("breakfast = %q", resp.Breakfast)
	}
	if resp.Calories != "2000 ккал" {
		t.Errorf("calories = %q", resp.Calories)
	}
	if len(resp.Snacks) != 2 {
		t.Errorf("snacks count = %d", len(resp.Snacks))
	}
}

func TestParseNutritionResponse_MissingCalories(t *testing.T) {
	raw := `{"breakfast": "Каша", "lunch": "Суп", "dinner": "Рыба"}`
	_, err := parseNutritionResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing calories")
	}
}

func TestParseMotivationResponse_Valid(t *testing.T) {
	raw := `{"text": "Давай, Алексей! 💪 Сегодня день ног, а не диванчика!"}`
	resp, err := parseMotivationResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text == "" {
		t.Fatal("expected non-empty text")
	}
}

func TestParseMotivationResponse_EmptyText(t *testing.T) {
	raw := `{"text": ""}`
	_, err := parseMotivationResponse(raw)
	if err == nil {
		t.Fatal("expected error for empty text")
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
