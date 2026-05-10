.PHONY: up down logs test test-go test-python test-integration clean build run dev migrate-up migrate-down migrate-new install-tools

# Start all services
up:
	docker-compose up --build -d

# Stop all services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Run all tests
test: test-go test-python test-integration

# Run Go unit tests
test-go:
	cd services/go-gateway && go test ./...

# Run Python unit tests
test-python:
	cd services/python-ai && python -m pytest tests/ -v

# Run integration tests (requires services to be running)
test-integration:
	cd integration-tests && python -m pytest -v

# Build the server binary
build:
	cd services/go-gateway && go build -o bin/server ./cmd/server

# Run the server without hot reload
run:
	cd services/go-gateway && go run ./cmd/server

# Run the server with live reloading (requires air)
dev:
	cd services/go-gateway && air

# Database migrations
migrate-up:
	cd services/go-gateway && go run ./cmd/server migrate up

migrate-down:
	cd services/go-gateway && go run ./cmd/server migrate down

migrate-new:
	@read -p "Migration name: " name; \
	cd services/go-gateway && go run ./cmd/server migrate create $$name

# Install dev tools
install-tools:
	go install github.com/air-verse/air@latest

# Clean everything (volumes + images)
clean:
	docker-compose down -v --rmi local
