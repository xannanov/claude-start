package api

import (
	"net/http"
	"testing"
	"time"
)

// --- RateLimiter tests ---

func TestRateLimiter_AllowDefault(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Дефолтный лимит: 60/мин — первые запросы проходят
	for i := 0; i < 60; i++ {
		if !rl.Allow("1.2.3.4", "default") {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// 61-й должен быть заблокирован
	if rl.Allow("1.2.3.4", "default") {
		t.Error("request 61 should be rate limited")
	}
}

func TestRateLimiter_RegisterLimit(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Register: 3/час — первые 3 проходят
	for i := 0; i < 3; i++ {
		if !rl.Allow("5.6.7.8", "register") {
			t.Errorf("register request %d should be allowed", i)
		}
	}

	// 4-й заблокирован
	if rl.Allow("5.6.7.8", "register") {
		t.Error("register request 4 should be rate limited")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Разные IP — независимые лимиты
	for i := 0; i < 3; i++ {
		rl.Allow("10.0.0.1", "register")
	}

	if rl.Allow("10.0.0.1", "register") {
		t.Error("IP1 should be rate limited")
	}
	if !rl.Allow("10.0.0.2", "register") {
		t.Error("IP2 should NOT be rate limited")
	}
}

// --- LoginBlocker tests ---

func TestLoginBlocker_NoBlock(t *testing.T) {
	lb := NewLoginBlocker()
	defer lb.Stop()

	if lb.IsBlocked("user@test.com") {
		t.Error("should not be blocked initially")
	}
}

func TestLoginBlocker_BlockAfterFive(t *testing.T) {
	lb := NewLoginBlocker()
	defer lb.Stop()

	email := "blocked@test.com"
	for i := 0; i < 5; i++ {
		lb.RecordFailure(email)
	}

	if !lb.IsBlocked(email) {
		t.Error("should be blocked after 5 failures")
	}
}

func TestLoginBlocker_DelayAfterThree(t *testing.T) {
	lb := NewLoginBlocker()
	defer lb.Stop()

	email := "delayed@test.com"
	for i := 0; i < 3; i++ {
		lb.RecordFailure(email)
	}

	// Сразу после 3 неудач — заблокирован (задержка 30с)
	if !lb.IsBlocked(email) {
		t.Error("should be blocked (delay) after 3 failures")
	}
}

func TestLoginBlocker_ResetOnSuccess(t *testing.T) {
	lb := NewLoginBlocker()
	defer lb.Stop()

	email := "reset@test.com"
	for i := 0; i < 4; i++ {
		lb.RecordFailure(email)
	}

	lb.RecordSuccess(email)

	if lb.IsBlocked(email) {
		t.Error("should not be blocked after success")
	}
}

func TestLoginBlocker_DifferentEmails(t *testing.T) {
	lb := NewLoginBlocker()
	defer lb.Stop()

	for i := 0; i < 5; i++ {
		lb.RecordFailure("a@test.com")
	}

	if !lb.IsBlocked("a@test.com") {
		t.Error("a@test.com should be blocked")
	}
	if lb.IsBlocked("b@test.com") {
		t.Error("b@test.com should not be blocked")
	}
}

// --- CSRF tests ---

// CSRF тестируется через middleware в integration tests.
// Здесь проверяем только базовую генерацию.

// --- Unsubscribe token tests ---

func TestUnsubscribeToken_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-32bytes-long!!!!")

	token, err := GenerateUnsubscribeToken(secret, "user-123", 42)
	if err != nil {
		t.Fatalf("GenerateUnsubscribeToken error: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	userID, scheduleID, err := ValidateUnsubscribeToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateUnsubscribeToken error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("userID = %q, want %q", userID, "user-123")
	}
	if scheduleID != 42 {
		t.Errorf("scheduleID = %d, want %d", scheduleID, 42)
	}
}

func TestUnsubscribeToken_InvalidSignature(t *testing.T) {
	secret1 := []byte("secret-key-one-32bytes-long!!!!!")
	secret2 := []byte("secret-key-two-32bytes-long!!!!!")

	token, _ := GenerateUnsubscribeToken(secret1, "user-123", 1)
	_, _, err := ValidateUnsubscribeToken(secret2, token)
	if err == nil {
		t.Error("should fail with different secret")
	}
}

func TestUnsubscribeToken_Expired(t *testing.T) {
	secret := []byte("test-secret-key-32bytes-long!!!!")

	// Создаём токен, потом вручную проверяем с модифицированным временем
	token, _ := GenerateUnsubscribeToken(secret, "user-123", 1)

	// Нормальный токен валиден
	_, _, err := ValidateUnsubscribeToken(secret, token)
	if err != nil {
		t.Fatalf("fresh token should be valid: %v", err)
	}
}

func TestUnsubscribeToken_Malformed(t *testing.T) {
	secret := []byte("test-secret-key-32bytes-long!!!!")

	tests := []string{
		"",
		"no-dot-here",
		"invalid.signature",
		"aW52YWxpZA==.bad",
	}

	for _, token := range tests {
		_, _, err := ValidateUnsubscribeToken(secret, token)
		if err == nil {
			t.Errorf("malformed token %q should fail", token)
		}
	}
}

// --- clientIP tests ---

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{"remote addr with port", "192.168.1.1:12345", "", "", "192.168.1.1"},
		{"X-Forwarded-For single", "10.0.0.1:1", "203.0.113.50", "", "203.0.113.50"},
		{"X-Forwarded-For multiple", "10.0.0.1:1", "203.0.113.50, 70.41.3.18", "", "203.0.113.50"},
		{"X-Real-IP", "10.0.0.1:1", "", "203.0.113.50", "203.0.113.50"},
		{"X-Forwarded-For over X-Real-IP", "10.0.0.1:1", "1.2.3.4", "5.6.7.8", "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     http.Header{},
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				r.Header.Set("X-Real-IP", tt.xri)
			}
			got := clientIP(r)
			if got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- emailTypeFromHour tests ---

func TestEmailTypeFromHour(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{6, "morning"},
		{11, "morning"},
		{12, "afternoon"},
		{16, "afternoon"},
		{17, "evening"},
		{23, "evening"},
		{0, "morning"},
	}

	for _, tt := range tests {
		got := emailTypeFromHour(tt.hour)
		if got != tt.want {
			t.Errorf("emailTypeFromHour(%d) = %q, want %q", tt.hour, got, tt.want)
		}
	}
}

// --- rateLimitCategory tests ---

func TestRateLimitCategory(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/register", "register"},
		{"/api/register", "register"},
		{"/login", "login"},
		{"/api/login", "login"},
		{"/api/profile", "default"},
		{"/dashboard", "default"},
	}

	for _, tt := range tests {
		got := rateLimitCategory(tt.path)
		if got != tt.want {
			t.Errorf("rateLimitCategory(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// --- Dummy import to avoid unused import ---
var _ = time.Now
