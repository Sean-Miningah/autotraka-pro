# go-gateway

`go-gateway` is a Go-based API gateway for WhatsApp messaging. It provides webhook ingestion, message routing, and outbound message delivery via the Meta WhatsApp Business API. It is built for observability, reliability, and easy local development.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| HTTP Router | [Chi](https://github.com/go-chi/chi) v5 |
| Configuration | [Viper](https://github.com/spf13/viper) (env vars + `.env`) |
| Logging | Go stdlib `log/slog` with OTel trace context injection |
| Tracing | [OpenTelemetry](https://opentelemetry.io) (OTLP/gRPC, BatchSpanProcessor) |
| Database ORM | [sqlx](https://github.com/jmoiron/sqlx) |
| Postgres Driver | [pgx](https://github.com/jackc/pgx) v5 |
| DB Migrations | [sql-migrate](https://github.com/rubenv/sql-migrate) |
| Live Reload | [Air](https://github.com/air-verse/air) |
| Build Tool | Make |

---

## Prerequisites

- Go **1.26+**
- Postgres **15+**
- (Optional) [NATS](https://nats.io/) for event bus
- (Optional) [Redis](https://redis.io/) for caching/sessions
- (Optional) OTel Collector for trace export

---

## Quick Start

### 1. Environment Setup

Copy the example environment file and adjust values as needed:

```bash
cp .env.example .env
```

Key variables (see `.env.example` for the full list):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `ENV` | `development` | Environment mode (`development` or `production`) |
| `DATABASE_URL` | `postgres://devuser:devpass@localhost:5432/wacrm?sslmode=disable` | Postgres connection string |
| `META_BASE_URL` | `http://localhost:1080` | Meta WhatsApp API base URL |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | OTel sampling strategy |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4317` | OTLP/gRPC collector endpoint |

### 2. Install Dev Tools

```bash
make install-tools
```

This installs `air` and `sql-migrate` globally.

### 3. Run Database Migrations

See the [Database Migrations](#database-migrations) section below.

### 4. Run in Dev Mode (Live Reload)

```bash
make dev
```

Air watches Go source files and `.env` and automatically rebuilds and restarts the server on change.

The binary is built to `./tmp/main` and ignored from version control.

### 5. Run Without Hot Reload

```bash
make run
```

### 6. Run Tests

```bash
make test
```

---

## Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Viper-based configuration loading
│   ├── db/              # sqlx DB pool + Querier interface
│   ├── eventbus/        # NATS JetStream publisher/subscriber (TODO)
│   ├── health/          # Health check handlers
│   ├── log/             # Context-attached slog with OTel trace injection
│   ├── messaging/       # WhatsApp webhook + Meta API client
│   └── telemetry/       # OTel tracer provider setup
├── migrations/          # sql-migrate migration files
├── pkg/                 # Shared public packages
├── .air.toml            # Air live-reload configuration
├── .env.example         # Environment variable template
├── dbconfig.yml         # sql-migrate configuration
├── Dockerfile           # Production container image
├── go.mod / go.sum      # Go module files
├── Makefile             # Build, dev, test, and migration commands
└── README.md            # This file
```

---

## Database Migrations

Migrations are managed by `sql-migrate` and configured via `dbconfig.yml`.

### Install sql-migrate

```bash
go install github.com/rubenv/sql-migrate/...@latest
```

### Run Migrations

```bash
# Apply pending migrations
make migrate-up

# Roll back the last migration
make migrate-down
```

### Create a New Migration

```bash
make migrate-new
# Enter the migration name when prompted, e.g., create_messages_table
```

This generates a new timestamped `.sql` file under `migrations/` with `Up` and `Down` sections.

### Manual sql-migrate Commands

```bash
# Apply migrations
sql-migrate up

# Roll back
sql-migrate down

# Status
sql-migrate status
```

The `dbconfig.yml` references the `DATABASE_URL` environment variable, so ensure it is exported or present in your `.env`.

---

## Configuration

Configuration is loaded in the following priority (highest to lowest):

1. **Environment variables**
2. **`.env` file** (loaded only when `ENV=development`)
3. **Built-in defaults** (see `internal/config/config.go`)

In **production**, `.env` is not loaded. All values must be injected by the deployment platform (12-factor app compliance).

### OpenTelemetry Resource Attributes

OTel service identity is configured purely through environment variables:

- `OTEL_SERVICE_NAME` — e.g., `go-gateway`
- `OTEL_RESOURCE_ATTRIBUTES` — e.g., `service.version=0.1.0,deployment.environment=production`
- `OTEL_TRACES_SAMPLER` — `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`
- `OTEL_TRACES_SAMPLER_ARG` — e.g., `0.01` for 1% sampling

---

## Observability

### Logging

- **Development**: Pretty-printed colored text to stdout.
- **Production**: Structured JSON with `trace_id`, `span_id`, and `severity`.

Logs automatically inherit trace context from the active OTel span. Use `log.FromContext(ctx)` to retrieve the request-scoped logger in handlers and repositories.

### Tracing

- **Incoming HTTP**: Every request through Chi is wrapped by `otelhttp` middleware.
- **Outgoing HTTP**: `MetaClient` uses `otelhttp.NewTransport` so calls to Meta's API appear as child spans.
- **Database**: `otelsql` wraps the `pgx` driver. Query spans are emitted when sampling is active.
- **Sampler**: `ParentBased` by default, overridable via `OTEL_TRACES_SAMPLER`.

---

## Docker

### Build Production Image

```bash
docker build -t go-gateway:latest .
```

### Run Container

```bash
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://... \
  -e OTEL_SERVICE_NAME=go-gateway \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317 \
  go-gateway:latest
```

The Dockerfile is multi-stage: a Go builder compiles the binary, then a minimal Alpine image runs it.

### Docker Compose (Example)

A minimal `docker-compose.yml` for local development:

```yaml
version: "3.9"
services:
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    depends_on:
      - postgres

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: devuser
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: wacrm
    ports:
      - "5432:5432"
```

---

## Deployment Notes

- **Environment**: Ensure `ENV=production` so `.env` is not loaded.
- **Secrets**: Inject `DATABASE_URL` and `META_BASE_URL` via your platform's secret management (e.g., Kubernetes secrets, AWS Secrets Manager).
- **OTel Collector**: Deploy an OTel Collector sidecar or service in your cluster. Point `OTEL_EXPORTER_OTLP_ENDPOINT` to it.
- **Graceful Shutdown**: The server handles `SIGTERM`/`SIGINT` by draining in-flight requests, flushing spans, and closing the DB pool. Ensure your orchestrator (K8s, ECS, etc.) provides adequate `terminationGracePeriodSeconds`.
- **Sampling**: Use `OTEL_TRACES_SAMPLER=traceidratio` and `OTEL_TRACES_SAMPLER_ARG=0.01` in high-traffic production environments to minimize overhead.
- **Health Checks**: The `/health` and `/ping` endpoints are exposed for load balancer health probes.

---

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile binary to `bin/server` |
| `make run` | Run server directly (`go run`) |
| `make dev` | Run with Air live reload |
| `make test` | Run Go tests |
| `make clean` | Remove `bin/` and `tmp/` |
| `make migrate-up` | Apply DB migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-new` | Create a new migration (interactive) |
| `make install-tools` | Install `air` and `sql-migrate` |

---

## License

Private / All Rights Reserved
