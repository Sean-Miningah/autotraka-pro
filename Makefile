.PHONY: up down logs test test-go test-python test-integration clean

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
	cd go-gateway && go test ./...

# Run Python unit tests
test-python:
	cd python-ai && python -m pytest tests/ -v

# Run integration tests (requires services to be running)
test-integration:
	cd integration-tests && python -m pytest -v

# Clean everything (volumes + images)
clean:
	docker-compose down -v --rmi local
