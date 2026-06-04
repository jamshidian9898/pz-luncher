package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting for Steam API calls
// This prevents API bans and handles 429 responses gracefully
type RateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64   // tokens per second
	lastRefill   time.Time
	minInterval  time.Duration // minimum time between requests
	lastRequest  time.Time
}

// NewRateLimiter creates a new rate limiter
// Default: 10 requests per second with burst of 20
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		tokens:      20,
		maxTokens:   20,
		refillRate:  10,
		lastRefill:  time.Now(),
		minInterval: 100 * time.Millisecond,
	}
}

// WithSteamLimits configures for Steam Web API limits
// Steam allows ~100,000 requests per day = ~1.15 per second
func (r *RateLimiter) WithSteamLimits() *RateLimiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maxTokens = 10
	r.tokens = 10
	r.refillRate = 1.0  // 1 per second = conservative
	r.minInterval = 1 * time.Second
	return r
}

// Acquire blocks until a token is available
// Returns context error if cancelled
func (r *RateLimiter) Acquire(ctx context.Context) error {
	for {
		if err := r.tryAcquire(); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Try again
		}
	}
}

// tryAcquire attempts to acquire a token without blocking
func (r *RateLimiter) tryAcquire() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Refill tokens based on time elapsed
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now

	// Check minimum interval
	if now.Sub(r.lastRequest) < r.minInterval {
		return fmt.Errorf("rate limited: min interval not met")
	}

	// Check token availability
	if r.tokens < 1 {
		return fmt.Errorf("rate limited: no tokens available")
	}

	// Consume token
	r.tokens--
	r.lastRequest = now
	return nil
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetStats returns current rate limiter statistics
func (r *RateLimiter) GetStats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]interface{}{
		"tokens":      r.tokens,
		"maxTokens":   r.maxTokens,
		"refillRate":  r.refillRate,
		"minInterval": r.minInterval.String(),
	}
}
