package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Rishi2804/pokepedia-api-v2/internal/cache"
)

// bufferedWriter captures a handler's response instead of forwarding it.
// It deliberately does NOT embed http.ResponseWriter: nothing escapes to the
// client until the response is known to be complete and cacheable.
//
// This forgoes http.Flusher/http.Hijacker promotion. That's safe today
// because every handler goes through writeJSON, which encodes a complete DTO
// in one shot — no streaming, no SSE, no websockets. A future streaming
// endpoint must bypass this middleware.
type bufferedWriter struct {
	header      http.Header
	buf         bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedWriter() *bufferedWriter {
	return &bufferedWriter{header: make(http.Header), status: http.StatusOK}
}

func (b *bufferedWriter) Header() http.Header { return b.header }

func (b *bufferedWriter) WriteHeader(status int) {
	if !b.wroteHeader {
		b.status = status
		b.wroteHeader = true
	}
}

func (b *bufferedWriter) Write(p []byte) (int, error) {
	b.wroteHeader = true
	return b.buf.Write(p)
}

// flushTo replays the captured response onto the real writer and sets
// X-Cache so cache behavior is observable from the client side.
func (b *bufferedWriter) flushTo(w http.ResponseWriter, cacheStatus string) {
	dst := w.Header()
	for k, v := range b.header {
		dst[k] = v
	}
	dst.Set("X-Cache", cacheStatus)
	w.WriteHeader(b.status)
	_, _ = w.Write(b.buf.Bytes())
}

func cacheKey(r *http.Request) string {
	return "pp:" + cache.KeyVersion + ":resp:" + r.URL.EscapedPath()
}

// ResponseCache caches 200 JSON responses keyed by request path. Safe only
// for routes that are a pure function of their path — every route this is
// applied to is a path-only GET with no query params.
func ResponseCache(c *cache.Cache) func(http.Handler) http.Handler {
	if !c.Enabled() {
		return func(next http.Handler) http.Handler { return next }
	}

	var sf singleflight.Group

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := cacheKey(r)

			if body, ok := c.Get(r.Context(), key); ok {
				h := w.Header()
				h.Set("Content-Type", "application/json")
				h.Set("X-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
				return
			}

			ch := sf.DoChan(key, func() (any, error) {
				// The leader must not inherit the triggering client's
				// cancellation — if that one client disconnects, every
				// follower waiting on this call would inherit a truncated
				// response. Detach, then re-bound with our own deadline.
				lctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
				defer cancel()
				return capture(c, next, r.WithContext(lctx), key), nil
			})

			select {
			case res := <-ch:
				bw := res.Val.(*bufferedWriter)
				status := "MISS"
				if res.Shared {
					status = "MISS-COALESCED"
				}
				bw.flushTo(w, status)
			case <-r.Context().Done():
				// This follower's own client went away; DoChan's channel is
				// buffered, so abandoning it here leaks nothing.
				return
			}
		})
	}
}

// capture runs the handler against a buffering writer and stores the result
// if — and only if — it's a complete, valid 200 response.
func capture(c *cache.Cache, next http.Handler, r *http.Request, key string) *bufferedWriter {
	bw := newBufferedWriter()
	next.ServeHTTP(bw, r)

	body := bw.buf.Bytes()
	switch {
	case bw.status != http.StatusOK:
		// Never cache 4xx/5xx. Also pins the keyspace to real data — a
		// scanner hitting garbage paths can't inflate it.
	case r.Context().Err() != nil:
		// Client disconnected or the 30s handler timeout fired mid-encode:
		// the body is a truncated prefix even though status is already 200.
	case !json.Valid(body):
		// writeJSON (internal/handlers/health.go) discards json.Encode's
		// error, so a partial write still looks like a 200. json.Valid on a
		// couple hundred KB is sub-millisecond and only runs on a MISS —
		// cheap insurance against caching a corrupt body for the full TTL.
	default:
		c.Set(r.Context(), key, body)
	}
	return bw
}
