-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (tenant_id, channel_id, channel_type, event_id, raw_payload) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (tenant_id, channel_type, event_id) DO NOTHING RETURNING id, tenant_id, channel_id, channel_type, event_id, raw_payload, processed, created_at;

-- name: ListUnprocessedWebhookEvents :many
SELECT id, tenant_id, channel_id, channel_type, event_id, raw_payload, processed, created_at FROM webhook_events WHERE processed = false ORDER BY created_at ASC LIMIT $1;

-- name: MarkWebhookEventProcessed :exec
UPDATE webhook_events SET processed = true WHERE id = $1;

-- name: GetWebhookEventByID :one
SELECT id, tenant_id, channel_id, channel_type, event_id, raw_payload, processed, created_at FROM webhook_events WHERE id = $1;
