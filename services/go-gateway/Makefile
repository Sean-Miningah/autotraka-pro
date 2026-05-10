.PHONY: build run dev test clean migrate-up migrate-down migrate-new sqlc-gen install-tools

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

# Database migrations (via CLI subcommand)
migrate-up:
	go run ./cmd/server migrate up

migrate-down:
	go run ./cmd/server migrate down

migrate-create:
	@read -p "Migration name: " name; \
	go run ./cmd/server migrate create $$name

# Generate sqlc code
sqlc-gen:
	sqlc generate

# Install dev tools
install-tools:
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
