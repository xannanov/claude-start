package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client — интерфейс для вызова AI API (для мокирования в тестах).
type Client interface {
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// GLMClient — реализация Client для z.ai GLM API (OpenAI-совместимый формат).
type GLMClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewGLMClient создаёт клиент для z.ai GLM API.
func NewGLMClient(apiKey, baseURL, model string) *GLMClient {
	return &GLMClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// ChatCompletion отправляет запрос к GLM API и возвращает ответ.
func (c *GLMClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP-запроса: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка HTTP-запроса к GLM API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа GLM API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GLM API вернул HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа GLM API: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("ошибка API GLM: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("GLM API вернул пустой ответ (нет choices)")
	}

	return &chatResp, nil
}
