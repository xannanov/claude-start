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

// parseCombinedResponse парсит единый ответ AI с тренировкой, питанием и мотивацией.
func parseCombinedResponse(raw string) (*aiCombinedResponse, error) {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}

	var resp aiCombinedResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if resp.Workout.Title == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует workout.title")
	}
	if resp.Workout.Duration == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует workout.duration")
	}
	if len(resp.Workout.Exercises) < 1 {
		return nil, fmt.Errorf("в ответе AI нет упражнений")
	}
	if resp.Nutrition.Breakfast == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует nutrition.breakfast")
	}
	if resp.Nutrition.Calories == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует nutrition.calories")
	}
	if resp.Motivation.Text == "" {
		return nil, fmt.Errorf("в ответе AI отсутствует motivation.text")
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
