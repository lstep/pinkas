.PHONY: all build dev test lint migrate sqlc docker

all: build

build:
	@echo "Building Go backend..."
	go build -tags fts5 -o server ./cmd/server
	@echo "Building frontend..."
	cd frontend && npm run build

DC := docker compose

dev:
	$(DC) up --build -d

dev-logs:
	$(DC) logs -f go-api collab nginx

dev-down:
	$(DC) down

docker-build:
	$(DC) build

docker-push:
	docker-compose push

fmt:
	go fmt ./...
