package services

import (
	"sync"
	"time"
)

type DistributedRateLimiter struct {
	window time.Duration

	mu      sync.Mutex
	buckets map[uint]rateBucket
}

type rateBucket struct {
	windowStart time.Time
	count       int
}

func NewDistributedRateLimiter(window time.Duration) *DistributedRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &DistributedRateLimiter{
		window:  window,
		buckets: make(map[uint]rateBucket),
	}
}

func (l *DistributedRateLimiter) Allow(keyID uint, limit int, now time.Time) bool {
	if keyID == 0 || limit == 0 {
		return true
	}
	if limit < 0 {
		return false
	}

	windowStart := now.UTC().Truncate(l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.gcLocked(now.UTC())

	current, ok := l.buckets[keyID]
	if !ok || !current.windowStart.Equal(windowStart) {
		l.buckets[keyID] = rateBucket{
			windowStart: windowStart,
			count:       1,
		}
		return true
	}
	if current.count >= limit {
		return false
	}
	current.count++
	l.buckets[keyID] = current
	return true
}

func (l *DistributedRateLimiter) gcLocked(now time.Time) {
	cutoff := now.Add(-2 * l.window)
	for keyID, bucket := range l.buckets {
		if bucket.windowStart.Before(cutoff) {
			delete(l.buckets, keyID)
		}
	}
}
