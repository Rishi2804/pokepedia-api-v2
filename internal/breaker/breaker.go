// Package breaker short-circuits calls to an optional dependency after
// repeated failures, so a dead backend costs ~0 per request instead of
// paying a timeout on every request. Shared by internal/cache (Redis) and
// internal/search (Elasticsearch) so both have one implementation and one
// set of semantics for what "degraded" means.
package breaker

import (
	"sync"
	"time"
)

type Breaker struct {
	threshold int
	cooldown  time.Duration

	mu        sync.Mutex
	failures  int
	openUntil time.Time
	probing   bool
}

func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown}
}

// Allow reports whether a call should be attempted. While open, it lets
// exactly one probe through per cooldown window so a recovered backend is
// noticed without hammering it.
func (b *Breaker) Allow() bool {
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

func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.probing = false
}

func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	b.probing = false
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
	}
}

func (b *Breaker) Healthy() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failures < b.threshold
}
