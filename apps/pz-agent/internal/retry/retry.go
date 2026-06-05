// Package retry provides a minimal exponential-backoff retry policy for
// Agent operations. Design constraints:
//
//   - No dependencies outside the standard library.
//   - Context-aware: cancellation stops retries immediately.
//   - Every attempt is logged via a caller-supplied log function so the
//     observer (B1) sees what is happening.
//   - Permanent errors (ErrPermanent) skip retries entirely.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrPermanent wraps an error and signals that it must not be retried.
// Use Permanent(err) to wrap, errors.Is / errors.As to unwrap.
type ErrPermanent struct{ Cause error }

func (e *ErrPermanent) Error() string { return fmt.Sprintf("permanent: %v", e.Cause) }
func (e *ErrPermanent) Unwrap() error { return e.Cause }

// Permanent marks err as a non-retryable error.
func Permanent(err error) error { return &ErrPermanent{Cause: err} }

// IsPermanent reports whether err is or wraps a permanent error.
func IsPermanent(err error) bool {
	var p *ErrPermanent
	return errors.As(err, &p)
}

// Policy defines retry behaviour.
type Policy struct {
	// MaxAttempts is the total number of attempts (1 = no retry).
	MaxAttempts int
	// Base is the delay before the second attempt.
	Base time.Duration
	// Max caps the computed delay.
	Max time.Duration
	// Jitter adds ±20% randomness when true (not implemented for simplicity).
}

// DefaultPolicy is sensible for Agent→Backend network calls.
var DefaultPolicy = Policy{
	MaxAttempts: 5,
	Base:        500 * time.Millisecond,
	Max:         30 * time.Second,
}

// HeartbeatPolicy is more lenient: heartbeats are fire-and-forget-ish.
var HeartbeatPolicy = Policy{
	MaxAttempts: 3,
	Base:        2 * time.Second,
	Max:         10 * time.Second,
}

// Do executes fn up to p.MaxAttempts times, backing off exponentially between
// attempts. It stops early if:
//   - fn returns nil (success)
//   - fn returns a Permanent error
//   - ctx is cancelled
//
// logf is called with a human-readable message on each transient failure; it
// may be nil.
func (p Policy) Do(ctx context.Context, opName string, logf func(string, ...any), fn func() error) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	var lastErr error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: context cancelled: %w", opName, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if IsPermanent(lastErr) {
			return fmt.Errorf("%s: %w", opName, lastErr)
		}

		if attempt == p.MaxAttempts {
			break
		}

		delay := backoff(p.Base, p.Max, attempt)
		logf("retry: %s attempt %d/%d failed (%v) — retrying in %v",
			opName, attempt, p.MaxAttempts, lastErr, delay)

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: context cancelled while waiting for retry: %w", opName, ctx.Err())
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("%s: all %d attempts failed: %w", opName, p.MaxAttempts, lastErr)
}

// backoff computes delay = base * 2^(attempt-1), capped at max.
func backoff(base, max time.Duration, attempt int) time.Duration {
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if d > max {
		return max
	}
	return d
}
