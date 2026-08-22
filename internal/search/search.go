// Package search is a fail-soft Elasticsearch client for site-wide search.
// Like internal/cache, every method is safe to call on a disabled client
// (empty URL) and on an unreachable cluster.
package search

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v9"

	"github.com/Rishi2804/pokepedia-api-v2/internal/breaker"
)

// ErrUnavailable covers a disabled client, an open breaker, a timeout, and a
// 5xx alike — callers cannot tell them apart and must not care; it means
// "fall back to Postgres".
var ErrUnavailable = errors.New("search: elasticsearch unavailable")

const (
	// Tighter than the cache breaker (5/30s): a degraded search is
	// user-visible and the Postgres fallback is genuinely usable, so there's
	// no reason to keep paying the timeout.
	breakerThreshold = 3
	breakerCooldown  = 20 * time.Second
	errLogInterval   = 30 * time.Second
)

const DefaultIndex = "pokepedia-search"

type Config struct {
	URL       string // empty = search disabled
	Index     string // alias name; "" => DefaultIndex
	OpTimeout time.Duration
}

type Stats struct {
	Enabled bool
	Healthy bool
	Queries int64
	Errors  int64
}

type Client struct {
	es    *elasticsearch.TypedClient // nil => disabled
	index string
	opTO  time.Duration
	br    *breaker.Breaker

	queries, errs atomic.Int64

	logMu      sync.Mutex
	lastErrLog time.Time
}

// New returns a usable *Client in every case. It returns a non-nil error
// only for a malformed config — an unreachable cluster is a degraded
// search, not a failed boot, and is handled at request time by the breaker.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return &Client{}, nil
	}

	es, err := elasticsearch.NewTyped(
		elasticsearch.WithAddresses(cfg.URL),
		elasticsearch.WithRetry(2, 502, 503, 504),
	)
	if err != nil {
		return nil, err
	}

	index := cfg.Index
	if index == "" {
		index = DefaultIndex
	}
	opTO := cfg.OpTimeout
	if opTO <= 0 {
		opTO = 800 * time.Millisecond
	}

	return &Client{
		es:    es,
		index: index,
		opTO:  opTO,
		br:    breaker.New(breakerThreshold, breakerCooldown),
	}, nil
}

// Enabled is nil-receiver safe.
func (c *Client) Enabled() bool {
	return c != nil && c.es != nil
}

// Ping is used once at boot and by /healthz. Never fatal to the caller, and
// deliberately does not touch the breaker — a boot-time ping shouldn't trip
// a breaker that request-time queries haven't opened yet.
func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	ok, err := c.es.Ping().Do(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnavailable
	}
	return nil
}

func (c *Client) Stats() Stats {
	if !c.Enabled() {
		return Stats{}
	}
	return Stats{
		Enabled: true,
		Healthy: c.br.Healthy(),
		Queries: c.queries.Load(),
		Errors:  c.errs.Load(),
	}
}

func (c *Client) onErr(err error) {
	c.br.Failure()
	c.errs.Add(1)

	c.logMu.Lock()
	defer c.logMu.Unlock()
	if time.Since(c.lastErrLog) < errLogInterval {
		return
	}
	c.lastErrLog = time.Now()
	log.Printf("search: elasticsearch error (rate-limited log): %v", err)
}
