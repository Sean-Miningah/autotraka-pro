# ADR 0001: Mockoon for Meta API Mocking

## Status

Accepted

## Context

The `go-gateway` service integrates with the Meta Business API (WhatsApp, Facebook Messenger, Instagram) and the `python-ai` service integrates with CRM APIs (Salesforce, HubSpot). We need a mock server for:

1. **Local development** — so engineers can run the full stack without live Meta/CRM credentials.
2. **CI tests** — so integration tests run deterministically without external network dependencies.

The existing `docker-compose.yml` declared a `mockserver` service (MockServer 5.15.0) with volume mounts to `mockserver/` and `mockserver/expectations`, but those directories never existed in the repo. The service was effectively dead code.

### Alternatives considered

- **Stoplight Prism** — schema-driven mocking using the official Meta OpenAPI v23.0 spec. Rejected because Prism's static mock responses are too generic for our test expectations, and custom overrides require significant work. Also, Prism has no built-in webhook/callback simulation.
- **MockServer (keep existing)** — Rejected because no expectations were ever written, and the JSON-based expectation format is cumbersome compared to Mockoon's route-per-endpoint model.
- **Lean Mockoon routes** — only define the ~6 routes our code actually hits. Rejected by the team in favor of importing the full v23.0 schema so every Meta endpoint is technically available for future expansion.

### Trade-offs

- **Request validation**: Prism validates outgoing requests against the OpenAPI schema; Mockoon does not. We accept this trade-off because our Go unit tests already assert exact request shapes.
- **File size**: The full imported schema produces a large environment JSON (~hundreds of KB). We accept this for the benefit of not needing a build step to merge schemas.
- **CRM + Meta in one container**: `python-ai` also needs CRM mocks. Adding 2 lightweight CRM stubs to the same Mockoon instance keeps `docker-compose.yml` simple.

## Decision

Use **Mockoon CLI** (`mockoon/cli:latest`) as the single mock container on port 1080.

1. Import the full Meta Business API v23.0 OpenAPI schema into a Mockoon environment.
2. Override the ~6 outbound routes our code actually hits with exact, deterministic responses.
3. Add webhook simulation routes that POST pre-signed payloads to our webhook handlers (HMAC pre-computed with a shared dev secret).
4. Add 2 lightweight CRM stub routes for `python-ai`.
5. Store the environment JSON at `mockoon/meta-mock.json`.
6. Update `docker-compose.yml` to replace the old `mockserver` service with `mockoon/cli`.

## Consequences

- Teammates can run `docker-compose up` without installing any mock tooling; the environment is self-contained.
- CI integration tests run against realistic Meta API shapes without network calls.
- Webhook end-to-end tests can be run at the container level (Mockoon fires webhooks → gateway processes → DB + NATS) rather than only via `httptest`.
- The large JSON file is reviewable but noisy in diffs. If Mockoon releases a new version, re-importing the schema may produce a large diff.
- If Meta deprecates v23.0, we must re-import a newer schema and update our code accordingly.
