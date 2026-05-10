-- name: CreateChannel :one
INSERT INTO channels (tenant_id, name, channel_type, config, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, tenant_id, name, channel_type, config, status, created_at, updated_at;

-- name: GetChannelByID :one
SELECT id, tenant_id, name, channel_type, config, status, created_at, updated_at FROM channels WHERE id = $1;

-- name: ListChannelsByTenant :many
SELECT id, tenant_id, name, channel_type, config, status, created_at, updated_at FROM channels WHERE tenant_id = $1;

-- name: ListChannelsByTenantAndType :many
SELECT id, tenant_id, name, channel_type, config, status, created_at, updated_at FROM channels WHERE tenant_id = $1 AND channel_type = $2;
