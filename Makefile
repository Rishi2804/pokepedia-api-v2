include .env
export

.PHONY: run build sqlc-generate migrate-up migrate-down migrate-force migrate-version docker-up docker-down cache-flush index index-dry es-health es-aliases scrape scrape-dry scrape-emit

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

sqlc-generate:
	sqlc generate

migrate-up:
	migrate -path db/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path db/migrations -database "$$DATABASE_URL" down 1

migrate-force:
	migrate -path db/migrations -database "$$DATABASE_URL" force 1

migrate-version:
	migrate -path db/migrations -database "$$DATABASE_URL" version

docker-up:
	docker compose up -d

docker-down:
	docker compose down

cache-flush:
	docker compose exec -T redis redis-cli FLUSHDB

index:
	go run ./cmd/indexer

index-dry:
	go run ./cmd/indexer -dry-run

scrape:
	go run ./cmd/scrape

scrape-dry:
	go run ./cmd/scrape -dry-run

scrape-emit:
	go run ./cmd/scrape -emit db/migrations/000008_descriptions.up.sql

es-health:
	curl -s http://localhost:9200/_cluster/health?pretty

es-aliases:
	curl -s 'http://localhost:9200/_cat/aliases?v'