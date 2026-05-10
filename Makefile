include .env
export
APP_PATH=cmd/server/main.go

run:
	go run $(APP_PATH)

migrate-up:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

docker-deps-up:
	docker compose up -d postgres redis

docker-migrate:
	docker compose run --rm migrate

docker-app-up:
	docker compose up -d --build app

docker-up: docker-deps-up docker-migrate docker-app-up

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app

frontend-install:
	npm --prefix frontend install

frontend-dev:
	npm --prefix frontend run dev
