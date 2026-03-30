package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// extractJSON извлекает JSON из текста ответа AI.
// Обрабатывает: чистый JSON, JSON в ```json...```, JSON в ```...```, JSON с лишним текстом.
func extractJSON(text string) (string, error) {
	text = strings.TrimSpace(text)

	// 1. Попробовать как чистый JSON
	if json.Valid([]byte(text)) {
		return text, nil
	}

	// 2. ```json ... ```
	re := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n\\s*```")
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}

	// 3. ``` ... ```
	re2 := regexp.MustCompile("(?s)```\\s*\\n(.*?)\\n\\s*```")
	if m := re2.FindStringSubmatch(text); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}

	// 4. Найти первый { и последний }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		candidate := text[start : end+1]
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("не удалось извлечь JSON из ответа AI: %s", truncate(text, 200))
}

// parseWorkoutResponse парсит ответ AI в модель тренировки.
func parseWorkoutResponse(raw string) (*aiWorkoutResponse, error) {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}

	var resp aiWorkoutResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON тренировки: %w", err)
	}

	if resp.Title == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует title тренировки")
	}
	if resp.Duration == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует duration тренировки")
	}
	if len(resp.Exercises) < 1 {
		return nil, fmt.Errorf("в ответе AI нет упражнений")
	}

	return &resp, nil
}

// parseNutritionResponse парсит ответ AI в модель питания.
func parseNutritionResponse(raw string) (*aiNutritionResponse, error) {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}

	var resp aiNutritionResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON питания: %w", err)
	}

	if resp.Breakfast == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует breakfast")
	}
	if resp.Lunch == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует lunch")
	}
	if resp.Dinner == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует dinner")
	}
	if resp.Calories == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует calories")
	}

	return &resp, nil
}

// parseMotivationResponse парсит ответ AI в мотивацию.
func parseMotivationResponse(raw string) (*aiMotivationResponse, error) {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}

	var resp aiMotivationResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON мотивации: %w", err)
	}

	if resp.Text == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует text мотивации")
	}

	return &resp, nil
}

// truncate обрезает строку до указанной длины.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
