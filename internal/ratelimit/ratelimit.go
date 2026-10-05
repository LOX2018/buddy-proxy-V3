// Package ratelimit provides a small fixed-window failure/request limiter
// for hardening unauthenticated endpoints and credential brute force.
package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count int
	reset time.Time
}

// Limiter 以 key（通常是客户端 IP）为维度做固定窗口计数：
// 窗口内超过 max 次触发拒绝，窗口结束自动恢复；
// Reset 用于鉴权成功后清零，避免正常用户被历史失败误伤。
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*entry
}

func New(max int, window time.Duration) *Limiter {
	if max <= 0 {
		max = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{max: max, window: window, hits: map[string]*entry{}}
}

// Allow 记录一次命中：窗口内累计已超过 max 时拒绝，并返回建议的
// Retry-After 时长（直到当前窗口结束）；否则放行。
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	e, exists := l.hits[key]
	if !exists || now.After(e.reset) {
		l.hits[key] = &entry{count: 1, reset: now.Add(l.window)}
		return true, 0
	}
	e.count++
	if e.count > l.max {
		return false, time.Until(e.reset)
	}
	return true, 0
}

// Reset 清除 key 的所有计数（鉴权通过后调用）。
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

// pruneLocked 仅在表过大时被动清一次过期项，避免长期驻留大量空闲 key。
func (l *Limiter) pruneLocked(now time.Time) {
	if len(l.hits) < 1024 {
		return
	}
	for k, e := range l.hits {
		if now.After(e.reset) {
			delete(l.hits, k)
		}
	}
}
