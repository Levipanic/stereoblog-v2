package httpapi

import (
	"sync"
	"time"
)

type fixedWindowLimiter struct {
	mu          sync.Mutex
	window      time.Duration
	max         int
	now         func() time.Time
	entries     map[string]limitEntry
	nextCleanup time.Time
}

type limitEntry struct {
	count int
	reset time.Time
}

func newFixedWindowLimiter(window time.Duration, max int) *fixedWindowLimiter {
	// ponytail: process-local state is sufficient for the single-instance VPS; use shared state only if deployment becomes multi-process.
	return &fixedWindowLimiter{window: window, max: max, now: time.Now, entries: map[string]limitEntry{}}
}

func (l *fixedWindowLimiter) allow(key string) (bool, time.Duration) {
	if l.max <= 0 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if !now.Before(l.nextCleanup) {
		for key, entry := range l.entries {
			if !now.Before(entry.reset) {
				delete(l.entries, key)
			}
		}
		l.nextCleanup = now.Add(l.window)
	}
	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.reset) {
		l.entries[key] = limitEntry{count: 1, reset: now.Add(l.window)}
		return true, 0
	}
	if entry.count >= l.max {
		return false, entry.reset.Sub(now)
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}
