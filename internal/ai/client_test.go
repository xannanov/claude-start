package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGLMClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected Authorization: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %q", r.Header.Get("Content-Type"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "glm-4.7-flash" {
			t.Errorf("unexpected model: %s", req.Model)
		}

		resp := ChatResponse{
			Choices: []ChatChoice{
				{Message: ChatMessage{Role: "assistant", Content: `{"text": "Привет!"}`}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGLMClient("test-key", server.URL, "glm-4.7-flash")
	resp, err := client.ChatCompletion(context.Background(), ChatRequest{
		Model:    "glm-4.7-flash",
		Messages: []ChatMessage{{Role: "user", Content: "тест"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != `{"text": "Привет!"}` {
		t.Errorf("unexpected content: %q", resp.Choices[0].Message.Content)
	}
}

func TestGLMClient_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewGLMClient("test-key", server.URL, "glm-4.7-flash")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "тест"}},
	})

	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
}

func TestGLMClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Error: &APIError{Message: "invalid api key"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGLMClient("bad-key", server.URL, "glm-4.7-flash")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "тест"}},
	})

	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestGLMClient_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{Choices: []ChatChoice{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGLMClient("test-key", server.URL, "glm-4.7-flash")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "тест"}},
	})

	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}
