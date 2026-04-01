package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	apiKey := os.Getenv("AI_API_KEY")
	baseURL := os.Getenv("AI_URL")
	model := os.Getenv("AI_MODEL")

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "Ты — персональный фитнес-тренер и диетолог. Верни только JSON без markdown:\n{\"workout\":{\"title\":\"...\",\"muscle_group\":\"chest\",\"duration\":\"40 мин\",\"description\":\"...\",\"exercises\":[{\"name\":\"Жим\",\"sets\":\"4 подхода\",\"reps\":\"10 раз\"}]},\"nutrition\":{\"breakfast\":\"Каша\",\"lunch\":\"Курица\",\"dinner\":\"Рыба\",\"snacks\":[\"Орехи\"],\"calories\":\"2400 ккал\",\"protein\":\"140 г\",\"fat\":\"70 г\",\"carbs\":\"280 г\",\"water_ml\":\"2800 мл\"},\"motivation\":{\"text\":\"Давай!\"}}",
			},
			{
				"role":    "user",
				"content": "Имя: Алексей, мужской, 30 лет, 180см, 80кг, цель: набор мышц, активный, Понедельник, утро",
			},
		},
		"temperature": 0.85,
		// "response_format": map[string]string{"type": "json_object"}, // ТЕСТ: отключено
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 90 * time.Second}
	req, _ := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	fmt.Printf("Запрос к %s, модель %s...\n", baseURL, model)
	start := time.Now()

	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("ОШИБКА (%v): %v\n", elapsed, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("HTTP %d (%v)\n\n", resp.StatusCode, elapsed.Round(time.Millisecond))

	// Красиво напечатать JSON
	var pretty any
	if json.Unmarshal(respBody, &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(string(respBody))
	}
}
