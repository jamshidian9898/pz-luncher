package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fastPolicy() Policy {
	return Policy{MaxAttempts: 3, Base: time.Millisecond, Max: 5 * time.Millisecond}
}

func TestDo_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), "op", nil, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), "op", nil, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustsAllAttempts(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), "op", nil, func() error {
		calls++
		return errors.New("always fails")
	})
	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts (MaxAttempts), got %d", calls)
	}
}

func TestDo_PermanentErrorSkipsRetries(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), "op", nil, func() error {
		calls++
		return Permanent(errors.New("do not retry me"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call for a permanent error, got %d", calls)
	}
}

func TestDo_ContextCancelledStopsRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := fastPolicy().Do(ctx, "op", nil, func() error {
		calls++
		cancel()
		return errors.New("transient")
	})
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call before cancellation is observed, got %d", calls)
	}
}

func TestIsPermanent(t *testing.T) {
	base := errors.New("boom")
	if IsPermanent(base) {
		t.Fatal("plain error should not be permanent")
	}
	if !IsPermanent(Permanent(base)) {
		t.Fatal("wrapped error should be permanent")
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	d := backoff(time.Second, 3*time.Second, 10)
	if d != 3*time.Second {
		t.Fatalf("expected backoff capped at 3s, got %v", d)
	}
}

func TestBackoff_DoublesEachAttempt(t *testing.T) {
	base := 100 * time.Millisecond
	max := time.Hour
	if got := backoff(base, max, 1); got != base {
		t.Fatalf("attempt 1: expected %v, got %v", base, got)
	}
	if got := backoff(base, max, 2); got != 2*base {
		t.Fatalf("attempt 2: expected %v, got %v", 2*base, got)
	}
	if got := backoff(base, max, 3); got != 4*base {
		t.Fatalf("attempt 3: expected %v, got %v", 4*base, got)
	}
}
