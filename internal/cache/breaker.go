package cache

import (
	"sync"
	"time"
)

// breaker short-circuits cache calls after repeated failures, so a dead Redis
// costs ~0 per request instead of paying OpTimeout on every request. Without
// this, a Redis outage would make every endpoint slower than having no cache
// at all.
type breaker struct {
	threshold int
	cooldown  time.Duration

	mu        sync.Mutex
	failures  int
	openUntil time.Time
	probing   bool
}

func newBreaker(threshold int, cooldown time.Duration) *breaker {
	return &breaker{threshold: threshold, cooldown: cooldown}
}

// allow reports whether a cache call should be attempted. While open, it lets
// exactly one probe through per cooldown window so a recovered Redis is
// noticed without hammering it.
func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.failures < b.threshold {
		return true
	}
	if time.Now().Before(b.openUntil) {
		return false
	}
	if b.probing {
		return false
	}
	b.probing = true
	return true
}

func (b *breaker) success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.probing = false
}

func (b *breaker) failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	b.probing = false
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
	}
}

func (b *breaker) healthy() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failures < b.threshold
}
