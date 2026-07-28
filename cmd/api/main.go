package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/config"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("starting server on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
