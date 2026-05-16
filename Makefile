.PHONY: up down logs infra-up infra-down test test-go test-python test-integration clean build run dev migrate-up migrate-down migrate-new install-tools ci-up ci-down ci-logs

# --- Development Commands ---

# Start infrastructure services (Postgres, Redis, NATS, Mockoon) for local development
infra-up:
	docker compose up -d

# Stop infrastructure services
infra-down:
	docker compose down

# Start all services for CI (includes go-gateway and python-ai)
ci-up:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml up --build -d

# Stop all CI services
ci-down:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml down

# View CI logs
ci-logs:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml logs -f

# View infrastructure logs
logs:
	docker compose logs -f

# --- Application Commands (run locally, infra must be running) ---

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

# Run the Python AI service locally
run-python-ai:
	cd services/python-ai && uvicorn app.main:app --host 0.0.0.0 --port 8081 --reload

# --- Database Migrations ---

migrate-up:
	cd services/go-gateway && go run ./cmd/server migrate up

migrate-down:
	cd services/go-gateway && go run ./cmd/server migrate down

migrate-new:
	@read -p "Migration name: " name; \
	cd services/go-gateway && go run ./cmd/server migrate create $$name

# --- Tools & Cleanup ---

# Install dev tools
install-tools:
	go install github.com/air-verse/air@latest

# Clean everything (volumes + images + containers)
clean:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml down -v --rmi local
