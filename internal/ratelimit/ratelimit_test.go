package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowShape(t *testing.T) {
	l := New(2, time.Minute)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("first hit should pass")
	}
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("second hit within window should pass")
	}
	if ok, retry := l.Allow("a"); ok || retry <= 0 {
		t.Fatalf("third hit should be rejected, got ok=%v retry=%v", ok, retry)
	}
	// 其它 key 不受影响。
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("different key should pass")
	}
}

func TestLimiterWindowRecovery(t *testing.T) {
	l := New(1, 20*time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("first should pass")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("second should be rejected")
	}
	time.Sleep(30 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("after window expiry should pass again")
	}
}

func TestLimiterReset(t *testing.T) {
	l := New(1, time.Minute)
	l.Allow("k")
	l.Allow("k") // reject
	l.Reset("k")
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("after Reset should pass")
	}
}

func TestLimiterZeroValue(t *testing.T) {
	l := New(0, 0)
	if ok, _ := l.Allow("x"); !ok {
		t.Fatal("New with zero args should still allow first hit")
	}
}
