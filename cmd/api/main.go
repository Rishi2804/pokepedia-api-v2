package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/cache"
	"github.com/Rishi2804/pokepedia-api-v2/internal/config"
	"github.com/Rishi2804/pokepedia-api-v2/internal/search"
	"github.com/Rishi2804/pokepedia-api-v2/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("connected to database successfully")

	c, err := cache.New(cache.Config{
		URL:       cfg.RedisURL,
		TTL:       cfg.CacheTTL,
		OpTimeout: cfg.CacheOpTimeout,
	})
	if err != nil {
		// Only a malformed REDIS_URL reaches here; an unreachable Redis does not.
		log.Fatalf("invalid REDIS_URL: %v", err)
	}
	defer c.Close()

	switch {
	case !c.Enabled():
		log.Println("response cache disabled (REDIS_URL not set)")
	default:
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := c.Ping(pingCtx); err != nil {
			// Non-fatal by design: Redis may come up after us, and the API
			// is fully functional without it in the meantime.
			log.Printf("warning: redis unreachable at boot, serving uncached: %v", err)
		} else {
			log.Printf("connected to redis successfully (ttl=%s)", cfg.CacheTTL)
		}
		cancel()
	}

	sc, err := search.New(search.Config{
		URL:       cfg.ElasticURL,
		Index:     cfg.ElasticIndex,
		OpTimeout: cfg.ElasticOpTimeout,
	})
	if err != nil {
		// Only a malformed ELASTIC_URL reaches here; an unreachable cluster does not.
		log.Fatalf("invalid ELASTIC_URL: %v", err)
	}

	switch {
	case !sc.Enabled():
		log.Println("search disabled (ELASTIC_URL not set); falling back to postgres")
	default:
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := sc.Ping(pingCtx); err != nil {
			// Non-fatal by design: Elasticsearch may come up after us, and
			// search degrades to the Postgres fallback in the meantime.
			log.Printf("warning: elasticsearch unreachable at boot, falling back to postgres: %v", err)
		} else {
			log.Println("connected to elasticsearch successfully")
		}
		cancel()
	}

	srv := server.New(pool, c, sc)

	log.Printf("starting server on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
