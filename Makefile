include .env
export

.PHONY: run build sqlc-generate migrate-up migrate-down migrate-force docker-up docker-down cache-flush

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