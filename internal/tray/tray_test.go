//go:build windows

package tray

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryUntilSucceedsOnFirstAttempt(t *testing.T) {
	var calls int32
	ok := retryUntil(50*time.Millisecond, 5*time.Millisecond, func() bool {
		atomic.AddInt32(&calls, 1)
		return true
	})
	if !ok {
		t.Fatal("expected success")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls=%d want 1", got)
	}
}

func TestRetryUntilRetriesUntilSuccess(t *testing.T) {
	var calls int32
	ok := retryUntil(500*time.Millisecond, 10*time.Millisecond, func() bool {
		return atomic.AddInt32(&calls, 1) >= 3
	})
	if !ok {
		t.Fatal("expected success")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls=%d want 3", got)
	}
}

func TestRetryUntilTimesOut(t *testing.T) {
	var calls int32
	start := time.Now()
	ok := retryUntil(40*time.Millisecond, 5*time.Millisecond, func() bool {
		atomic.AddInt32(&calls, 1)
		return false
	})
	if ok {
		t.Fatal("expected timeout failure")
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("returned before deadline: %s", elapsed)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("calls=%d want >=2", calls)
	}
}
