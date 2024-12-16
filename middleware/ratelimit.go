package middleware

import (
	"sync"
	"time"
)

type LoginAttempt struct {
	Count     int
	FirstTry  time.Time
	BlockedAt time.Time
}

type RateLimiter struct {
	attempts map[string]*LoginAttempt
	mu       sync.RWMutex
}

var Limiter = &RateLimiter{
	attempts: make(map[string]*LoginAttempt),
}

const (
	maxAttempts    = 5
	blockDuration  = 15 * time.Minute
	windowDuration = 5 * time.Minute
)

func (rl *RateLimiter) IsBlocked(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	// If blocked and block duration hasn't expired
	if !attempt.BlockedAt.IsZero() && time.Since(attempt.BlockedAt) < blockDuration {
		return true
	}

	return false
}

func (rl *RateLimiter) RecordAttempt(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	attempt, exists := rl.attempts[ip]

	if !exists {
		rl.attempts[ip] = &LoginAttempt{
			Count:    1,
			FirstTry: now,
		}
		return true
	}

	// Reset if window duration has passed
	if time.Since(attempt.FirstTry) > windowDuration {
		attempt.Count = 1
		attempt.FirstTry = now
		attempt.BlockedAt = time.Time{}
		return true
	}

	// Increment attempt count
	attempt.Count++

	// Block if exceeded max attempts
	if attempt.Count > maxAttempts {
		attempt.BlockedAt = now
		return false
	}

	return true
}

func (rl *RateLimiter) GetRemainingAttempts(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		return maxAttempts
	}

	if time.Since(attempt.FirstTry) > windowDuration {
		return maxAttempts
	}

	return maxAttempts - attempt.Count
}
