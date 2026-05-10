.PHONY: build run dev test clean migrate-up migrate-down migrate-new

# Build the server binary
build:
	go build -o bin/server ./cmd/server

# Run the server without hot reload
run:
	go run ./cmd/server

# Run the server with live reloading (requires air)
dev:
	air

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Database migrations
migrate-up:
	sql-migrate up

migrate-down:
	sql-migrate down

migrate-new:
	@read -p "Migration name: " name; \
	sql-migrate new $$name

# Install dev tools
install-tools:
	go install github.com/rubenv/sql-migrate/...@latest
	go install github.com/air-verse/air@latest
