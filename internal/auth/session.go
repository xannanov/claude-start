package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	// SessionCookieName — имя cookie для хранения токена сессии.
	SessionCookieName = "session_token"

	// sessionTokenBytes — длина токена в байтах (32 байта = 256 бит).
	sessionTokenBytes = 32

	// SessionTTL — время жизни сессии.
	SessionTTL = 24 * time.Hour

	// cleanupInterval — интервал очистки просроченных сессий.
	cleanupInterval = 10 * time.Minute
)

// Session хранит данные авторизованной сессии.
type Session struct {
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionManager управляет сессиями пользователей (in-memory).
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]Session // token -> Session
	done     chan struct{}
}

// NewSessionManager создаёт менеджер сессий и запускает фоновую очистку.
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]Session),
		done:     make(chan struct{}),
	}
	go sm.cleanupLoop()
	return sm
}

// Create создаёт новую сессию для пользователя и возвращает токен.
func (sm *SessionManager) Create(userID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	sm.mu.Lock()
	sm.sessions[token] = Session{
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionTTL),
	}
	sm.mu.Unlock()

	return token, nil
}

// Validate проверяет токен и возвращает сессию.
// Возвращает ошибку если токен не найден или истёк.
func (sm *SessionManager) Validate(token string) (Session, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[token]
	sm.mu.RUnlock()

	if !ok {
		return Session{}, fmt.Errorf("сессия не найдена")
	}
	if time.Now().After(session.ExpiresAt) {
		sm.Delete(token)
		return Session{}, fmt.Errorf("сессия истекла")
	}

	return session, nil
}

// Delete удаляет сессию по токену.
func (sm *SessionManager) Delete(token string) {
	sm.mu.Lock()
	delete(sm.sessions, token)
	sm.mu.Unlock()
}

// DeleteByUserID удаляет все сессии пользователя (logout everywhere).
func (sm *SessionManager) DeleteByUserID(userID string) {
	sm.mu.Lock()
	for token, session := range sm.sessions {
		if session.UserID == userID {
			delete(sm.sessions, token)
		}
	}
	sm.mu.Unlock()
}

// SetCookie устанавливает session cookie в HTTP-ответ.
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(SessionTTL.Seconds()),
	})
}

// ClearCookie удаляет session cookie.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// Stop останавливает фоновую очистку сессий.
func (sm *SessionManager) Stop() {
	close(sm.done)
}

// ActiveCount возвращает количество активных сессий.
func (sm *SessionManager) ActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.removeExpired()
		case <-sm.done:
			return
		}
	}
}

func (sm *SessionManager) removeExpired() {
	now := time.Now()
	sm.mu.Lock()
	for token, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, token)
		}
	}
	sm.mu.Unlock()
}

func generateToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ошибка генерации токена: %w", err)
	}
	return hex.EncodeToString(b), nil
}
