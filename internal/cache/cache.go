// Package cache provides a fail-open Redis-backed response cache. Every
// method is safe to call on a disabled cache (empty URL) — callers never
// need to branch on whether caching is turned on.
package cache

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyVersion is embedded in every cache key. Bump it in the same commit that
// changes a cached DTO's shape — old keys orphan instantly and age out via
// TTL, no flush required.
const KeyVersion = "v1"

const (
	breakerThreshold = 5
	breakerCooldown  = 30 * time.Second
	errLogInterval   = 30 * time.Second
)

type Config struct {
	URL          string
	TTL          time.Duration
	OpTimeout    time.Duration
	MaxEntrySize int // bodies larger than this are never stored; 0 = default (1MB)
}

type Stats struct {
	Enabled bool
	Healthy bool
	Hits    int64
	Misses  int64
	Errors  int64
}

type Cache struct {
	rdb      *redis.Client // nil => disabled
	ttl      time.Duration
	opTO     time.Duration
	maxEntry int
	breaker  *breaker

	hits, misses, errs atomic.Int64

	logMu      sync.Mutex
	lastErrLog time.Time
}

// New returns a usable *Cache in every case. It returns a non-nil error only
// for a malformed URL — an unreachable Redis is a degraded cache, not a
// failed boot, and is handled at request time by the breaker instead.
func New(cfg Config) (*Cache, error) {
	if cfg.URL == "" {
		return &Cache{}, nil
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	opTO := cfg.OpTimeout
	if opTO <= 0 {
		opTO = 150 * time.Millisecond
	}
	maxEntry := cfg.MaxEntrySize
	if maxEntry <= 0 {
		maxEntry = 1 << 20
	}

	return &Cache{
		rdb:      redis.NewClient(opts),
		ttl:      ttl,
		opTO:     opTO,
		maxEntry: maxEntry,
		breaker:  newBreaker(breakerThreshold, breakerCooldown),
	}, nil
}

// Enabled is nil-receiver safe.
func (c *Cache) Enabled() bool {
	return c != nil && c.rdb != nil
}

// Get returns (nil, false) for a miss, a disabled cache, an open breaker, or
// any Redis error. Callers cannot distinguish and must not care — a cache
// failure is always just a miss from the caller's perspective.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool) {
	if !c.Enabled() || !c.breaker.allow() {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(ctx, c.opTO)
	defer cancel()

	b, err := c.rdb.Get(ctx, key).Bytes()
	switch {
	case err == nil:
		c.breaker.success()
		c.hits.Add(1)
		return b, true
	case errors.Is(err, redis.Nil):
		c.breaker.success() // a clean miss proves Redis is alive
		c.misses.Add(1)
		return nil, false
	default:
		c.onErr(err)
		return nil, false
	}
}

// Set is fire-and-forget: it never blocks the response and detaches from the
// request context so a client disconnecting doesn't abort a write that's
// already worth doing.
func (c *Cache) Set(ctx context.Context, key string, val []byte) {
	if !c.Enabled() || !c.breaker.allow() || len(val) > c.maxEntry {
		return
	}

	bg := context.WithoutCancel(ctx)
	go func() {
		ctx, cancel := context.WithTimeout(bg, 2*time.Second)
		defer cancel()
		if err := c.rdb.Set(ctx, key, val, c.ttl).Err(); err != nil {
			c.onErr(err)
			return
		}
		c.breaker.success()
	}()
}

// Ping is used once at boot and by /healthz. Never fatal to the caller.
func (c *Cache) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Stats() Stats {
	if !c.Enabled() {
		return Stats{}
	}
	return Stats{
		Enabled: true,
		Healthy: c.breaker.healthy(),
		Hits:    c.hits.Load(),
		Misses:  c.misses.Load(),
		Errors:  c.errs.Load(),
	}
}

func (c *Cache) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Close()
}

func (c *Cache) onErr(err error) {
	c.breaker.failure()
	c.errs.Add(1)

	c.logMu.Lock()
	defer c.logMu.Unlock()
	if time.Since(c.lastErrLog) < errLogInterval {
		return
	}
	c.lastErrLog = time.Now()
	log.Printf("cache: redis error (rate-limited log): %v", err)
}
