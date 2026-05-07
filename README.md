# WhatsApp AI-CRM Communication Platform

> Local development environment for the WhatsApp AI-CRM platform.

## Quick Start

```bash
# Start all services
docker-compose up --build

# Or use Make
make up
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| go-gateway | 8080 | Go API Gateway (Chi) |
| python-ai | 8081 | Python AI Orchestration (FastAPI) |
| postgres | 5432 | Primary database (with pgvector) |
| redis | 6379 | Cache & sessions |
| nats | 4222 | Event bus |
| mockserver | 1080 | External API mocking (Meta, CRMs) |

## Testing

```bash
# Run all tests
make test

# Run Go tests only
make test-go

# Run Python tests only
make test-python

# Run integration tests
make test-integration
```

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full system specification.

## Mocking External APIs

We use [MockServer](https://www.mock-server.com/) for integration testing against external APIs (Meta WhatsApp, Salesforce, HubSpot).

Mock expectations are pre-loaded from `mockserver/expectations/` on startup.

### Adding New Expectations

1. Create a JSON file in `mockserver/expectations/`
2. Restart MockServer: `docker-compose restart mockserver`
3. Or push dynamically via the MockServer API:

```bash
curl -X PUT http://localhost:1080/mockserver/expectation \
  -H 'Content-Type: application/json' \
  -d @mockserver/expectations/my-new-api.json
```

## Development

```bash
# View logs
make logs

# Stop all services
make down

# Clean everything (volumes + images)
make clean
```
# autotraka-pro
