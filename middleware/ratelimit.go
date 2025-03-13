package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
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
	RegistrationLimiter  = NewRateLimiter(2, 15*time.Minute) // 2 registrations per 15 minutes per IP
)

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string]*LoginAttempt),
		window:   window,
		limit:    limit,
	}
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r *http.Request) string {
	// For local testing, always return a fixed string to ensure rate limiting works
	// This ensures we're not affected by different representations of localhost
	if strings.Contains(r.RemoteAddr, "127.0.0.1") ||
		strings.Contains(r.RemoteAddr, "::1") ||
		strings.Contains(r.RemoteAddr, "localhost") {
		return "LOCAL_TESTING_IP"
	}

	// Get IP from RemoteAddr
	ipPort := r.RemoteAddr
	ip, _, err := net.SplitHostPort(ipPort)
	if err != nil {
		ip = ipPort // Use the full string if SplitHostPort fails
	}

	// Check for proxy headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if clientIP != "" {
				ip = clientIP
			}
		}
	} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip = xri
	}

	log.Printf("Client IP identified as: %s", ip)
	return ip
}

// CheckLimit checks if the request should be rate limited
func (rl *RateLimiter) CheckLimit(r *http.Request) (int, time.Time, error) {
	// Get the real IP address
	ip := GetClientIP(r)
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Debug the state of attempts map
	log.Printf("Current attempts map state for IP %s: %+v", ip, rl.attempts[ip])

	// Clean up old attempts
	rl.cleanup(now)

	// Get or create attempt for this IP
	attempt, exists := rl.attempts[ip]
	if !exists {
		log.Printf("No existing attempts for IP %s, creating new entry", ip)
		attempt = &LoginAttempt{
			Count: 0,
			First: now,
		}
		rl.attempts[ip] = attempt
	}

	// Check if blocked
	if attempt.Count >= rl.limit {
		blockUntil := attempt.First.Add(rl.window)
		log.Printf("IP %s has reached limit. Count: %d, Limit: %d, Blocked until: %v",
			ip, attempt.Count, rl.limit, blockUntil)

		if now.Before(blockUntil) {
			return 0, blockUntil, nil
		}

		// Reset if window has passed
		log.Printf("Window has passed for IP %s, resetting count", ip)
		attempt.Count = 0
		attempt.First = now
	}

	// Increment attempt count
	attempt.Count++
	log.Printf("Incrementing count for IP %s to %d", ip, attempt.Count)

	remaining := rl.limit - attempt.Count
	nextReset := attempt.First.Add(rl.window)
	log.Printf("IP %s has %d attempts remaining until %v", ip, remaining, nextReset)

	return remaining, nextReset, nil
}

// cleanup removes old attempts
func (rl *RateLimiter) cleanup(now time.Time) {
	for ip, attempt := range rl.attempts {
		if now.Sub(attempt.First) > rl.window {
			log.Printf("Cleaning up expired attempt for IP %s", ip)
			delete(rl.attempts, ip)
		}
	}
}
