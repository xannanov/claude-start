package api

import (
	"sync"
	"time"
)

const (
	loginDelayThreshold = 3              // после 3 неудач — задержка
	loginBlockThreshold = 5              // после 5 неудач — блокировка
	loginDelayDuration  = 30 * time.Second
	loginBlockDuration  = 15 * time.Minute
)

type loginAttempt struct {
	failures    int
	blockedUtil time.Time
	lastFailure time.Time
}

// LoginBlocker отслеживает неудачные попытки входа по email.
type LoginBlocker struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
	done     chan struct{}
}

// NewLoginBlocker создаёт блокировщик с фоновой очисткой.
func NewLoginBlocker() *LoginBlocker {
	lb := &LoginBlocker{
		attempts: make(map[string]*loginAttempt),
		done:     make(chan struct{}),
	}
	go lb.cleanupLoop()
	return lb
}

// IsBlocked проверяет, заблокирован ли вход для данного email.
func (lb *LoginBlocker) IsBlocked(email string) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	a, ok := lb.attempts[email]
	if !ok {
		return false
	}

	now := time.Now()

	// Блокировка после 5 неудач
	if a.failures >= loginBlockThreshold && now.Before(a.blockedUtil) {
		return true
	}

	// Задержка после 3 неудач
	if a.failures >= loginDelayThreshold && now.Before(a.lastFailure.Add(loginDelayDuration)) {
		return true
	}

	return false
}

// RecordFailure фиксирует неудачную попытку входа.
func (lb *LoginBlocker) RecordFailure(email string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	a, ok := lb.attempts[email]
	if !ok {
		a = &loginAttempt{}
		lb.attempts[email] = a
	}

	a.failures++
	a.lastFailure = time.Now()

	if a.failures >= loginBlockThreshold {
		a.blockedUtil = time.Now().Add(loginBlockDuration)
	}
}

// RecordSuccess сбрасывает счётчик после успешного входа.
func (lb *LoginBlocker) RecordSuccess(email string) {
	lb.mu.Lock()
	delete(lb.attempts, email)
	lb.mu.Unlock()
}

// Stop останавливает фоновую очистку.
func (lb *LoginBlocker) Stop() {
	close(lb.done)
}

func (lb *LoginBlocker) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lb.removeStale()
		case <-lb.done:
			return
		}
	}
}

func (lb *LoginBlocker) removeStale() {
	cutoff := time.Now().Add(-1 * time.Hour)
	lb.mu.Lock()
	for email, a := range lb.attempts {
		if a.lastFailure.Before(cutoff) {
			delete(lb.attempts, email)
		}
	}
	lb.mu.Unlock()
}
