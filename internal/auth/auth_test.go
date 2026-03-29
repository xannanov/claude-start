package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Password tests ---

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if hash == "secret123" {
		t.Fatal("hash should not equal plaintext")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	hash1, _ := HashPassword("same_password")
	hash2, _ := HashPassword("same_password")
	if hash1 == hash2 {
		t.Error("two hashes of same password should differ (unique salt)")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	hash, _ := HashPassword("mypassword")
	if err := CheckPassword(hash, "mypassword"); err != nil {
		t.Errorf("CheckPassword should succeed for correct password: %v", err)
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, _ := HashPassword("mypassword")
	if err := CheckPassword(hash, "wrongpassword"); err == nil {
		t.Error("CheckPassword should fail for wrong password")
	}
}

func TestCheckPassword_Empty(t *testing.T) {
	hash, _ := HashPassword("mypassword")
	if err := CheckPassword(hash, ""); err == nil {
		t.Error("CheckPassword should fail for empty password")
	}
}

// --- Session tests ---

func TestSessionManager_CreateAndValidate(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	token, err := sm.Create("user-123")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
	if len(token) != sessionTokenBytes*2 { // hex encoding
		t.Errorf("token length = %d, want %d", len(token), sessionTokenBytes*2)
	}

	session, err := sm.Validate(token)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if session.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", session.UserID, "user-123")
	}
}

func TestSessionManager_ValidateInvalid(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	_, err := sm.Validate("nonexistent-token")
	if err == nil {
		t.Error("Validate should fail for nonexistent token")
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	token, _ := sm.Create("user-456")

	sm.Delete(token)

	_, err := sm.Validate(token)
	if err == nil {
		t.Error("Validate should fail after Delete")
	}
}

func TestSessionManager_DeleteByUserID(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	token1, _ := sm.Create("user-789")
	token2, _ := sm.Create("user-789")
	token3, _ := sm.Create("other-user")

	sm.DeleteByUserID("user-789")

	if _, err := sm.Validate(token1); err == nil {
		t.Error("token1 should be invalid after DeleteByUserID")
	}
	if _, err := sm.Validate(token2); err == nil {
		t.Error("token2 should be invalid after DeleteByUserID")
	}
	if _, err := sm.Validate(token3); err != nil {
		t.Error("token3 (other user) should still be valid")
	}
}

func TestSessionManager_ExpiredSession(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	token, _ := sm.Create("user-exp")

	// Вручную сделаем сессию просроченной.
	sm.mu.Lock()
	s := sm.sessions[token]
	s.ExpiresAt = time.Now().Add(-1 * time.Second)
	sm.sessions[token] = s
	sm.mu.Unlock()

	_, err := sm.Validate(token)
	if err == nil {
		t.Error("Validate should fail for expired session")
	}

	// Сессия должна быть удалена.
	if sm.ActiveCount() != 0 {
		t.Errorf("ActiveCount = %d, want 0 after expired validation", sm.ActiveCount())
	}
}

func TestSessionManager_ActiveCount(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	if sm.ActiveCount() != 0 {
		t.Errorf("initial ActiveCount = %d, want 0", sm.ActiveCount())
	}

	sm.Create("u1")
	sm.Create("u2")

	if sm.ActiveCount() != 2 {
		t.Errorf("ActiveCount = %d, want 2", sm.ActiveCount())
	}
}

func TestSessionManager_UniqueTokens(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := sm.Create("user")
		if err != nil {
			t.Fatalf("Create error on iteration %d: %v", i, err)
		}
		if tokens[token] {
			t.Fatalf("duplicate token on iteration %d", i)
		}
		tokens[token] = true
	}
}

// --- Middleware tests ---

func TestRequireAuth_NoToken(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	handler := RequireAuth(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	handler := RequireAuth(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "bad-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	token, _ := sm.Create("user-auth")

	var gotUserID string
	handler := RequireAuth(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUserID != "user-auth" {
		t.Errorf("userID = %q, want %q", gotUserID, "user-auth")
	}
}

func TestUserIDFromContext_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id := UserIDFromContext(req.Context()); id != "" {
		t.Errorf("UserIDFromContext = %q, want empty", id)
	}
}
