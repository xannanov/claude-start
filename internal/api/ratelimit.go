package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig задаёт параметры rate limiting для категории эндпоинтов.
type RateLimitConfig struct {
	Rate  rate.Limit
	Burst int
}

var (
	// register: 15 запросов/час
	rateLimitRegister = RateLimitConfig{Rate: rate.Every(time.Hour / 15), Burst: 15}
	// login: 10 запросов/15 минут
	rateLimitLogin = RateLimitConfig{Rate: rate.Every(15 * time.Minute / 10), Burst: 10}
	// остальные: 60 запросов/минуту
	rateLimitDefault = RateLimitConfig{Rate: rate.Every(time.Minute / 60), Burst: 60}
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter управляет rate limiting по IP и категории.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter // ключ: "category:ip"
	done     chan struct{}
}

// NewRateLimiter создаёт rate limiter с фоновой очисткой.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		done:     make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Allow проверяет, разрешён ли запрос для данного IP и категории.
func (rl *RateLimiter) Allow(ip, category string) bool {
	cfg := rateLimitDefault
	switch category {
	case "register":
		cfg = rateLimitRegister
	case "login":
		cfg = rateLimitLogin
	}

	key := category + ":" + ip

	rl.mu.Lock()
	lim, ok := rl.limiters[key]
	if !ok {
		lim = &ipLimiter{
			limiter: rate.NewLimiter(cfg.Rate, cfg.Burst),
		}
		rl.limiters[key] = lim
	}
	lim.lastSeen = time.Now()
	rl.mu.Unlock()

	return lim.limiter.Allow()
}

// Stop останавливает фоновую очистку.
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.removeStale()
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) removeStale() {
	cutoff := time.Now().Add(-1 * time.Hour)
	rl.mu.Lock()
	for key, lim := range rl.limiters {
		if lim.lastSeen.Before(cutoff) {
			delete(rl.limiters, key)
		}
	}
	rl.mu.Unlock()
}

// clientIP извлекает IP-адрес клиента из запроса.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Берём первый IP (клиентский)
		for i, ch := range xff {
			if ch == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Убираем порт из RemoteAddr
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
