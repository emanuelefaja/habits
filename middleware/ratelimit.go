package middleware

import (
	"net/http"
	"sync"
	"time"
)

// LoginAttempt represents a login attempt
type LoginAttempt struct {
	Count int
	First time.Time
}

// RateLimiter handles rate limiting for different endpoints
type RateLimiter struct {
	attempts map[string]*LoginAttempt
	mu       sync.RWMutex
	window   time.Duration
	limit    int
}

var (
	LoginLimiter         = NewRateLimiter(5, 15*time.Minute) // 5 attempts per 15 minutes
	PasswordResetLimiter = NewRateLimiter(3, 24*time.Hour)   // 3 attempts per 24 hours
)

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string]*LoginAttempt),
		window:   window,
		limit:    limit,
	}
}

// CheckLimit checks if the request should be rate limited
func (rl *RateLimiter) CheckLimit(r *http.Request) (int, time.Time, error) {
	ip := r.RemoteAddr
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clean up old attempts
	rl.cleanup(now)

	// Get or create attempt for this IP
	attempt, exists := rl.attempts[ip]
	if !exists {
		attempt = &LoginAttempt{
			Count: 0,
			First: now,
		}
		rl.attempts[ip] = attempt
	}

	// Check if blocked
	if attempt.Count >= rl.limit {
		blockUntil := attempt.First.Add(rl.window)
		if now.Before(blockUntil) {
			return 0, blockUntil, nil
		}
		// Reset if window has passed
		attempt.Count = 0
		attempt.First = now
	}

	// Increment attempt count
	attempt.Count++

	return rl.limit - attempt.Count, attempt.First.Add(rl.window), nil
}

// cleanup removes old attempts
func (rl *RateLimiter) cleanup(now time.Time) {
	for ip, attempt := range rl.attempts {
		if now.Sub(attempt.First) > rl.window {
			delete(rl.attempts, ip)
		}
	}
}
