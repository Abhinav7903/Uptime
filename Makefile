.PHONY: dev build test migrate-up migrate-down

dev:
	docker-compose up

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

build-api:
	go build -o bin/api cmd/api/main.go

build-worker:
	go build -o bin/worker cmd/worker/main.go
