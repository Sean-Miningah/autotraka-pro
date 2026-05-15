-- name: CreateBroadcast :one
INSERT INTO broadcasts (tenant_id, title, template_id, parameters, channel_id, status, scheduled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at;

-- name: GetBroadcastByID :one
SELECT id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at
FROM broadcasts WHERE id = $1 AND tenant_id = $2;

-- name: ListBroadcastsByTenant :many
SELECT id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at
FROM broadcasts WHERE tenant_id = $1
ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountBroadcastsByTenant :one
SELECT COUNT(*) FROM broadcasts WHERE tenant_id = $1;

-- name: UpdateBroadcast :one
UPDATE broadcasts SET
    title = $1,
    template_id = $2,
    parameters = $3,
    channel_id = $4,
    status = $5,
    scheduled_at = $6,
    updated_at = now()
WHERE id = $7 AND tenant_id = $8
RETURNING id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at;

-- name: UpdateBroadcastStatus :one
UPDATE broadcasts SET status = $1, started_at = $2, completed_at = $3, updated_at = now()
WHERE id = $4 AND tenant_id = $5
RETURNING id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at;

-- name: DeleteBroadcast :exec
DELETE FROM broadcasts WHERE id = $1 AND tenant_id = $2;

-- name: GetBroadcastByIDForWorker :one
SELECT id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at
FROM broadcasts WHERE id = $1;

-- name: ListScheduledBroadcastsReady :many
SELECT id, tenant_id, title, template_id, parameters, channel_id, status, scheduled_at, started_at, completed_at, created_at, updated_at
FROM broadcasts
WHERE status = 'scheduled' AND scheduled_at <= now()
ORDER BY scheduled_at ASC;

-- Recipients
-- name: AddBroadcastRecipients :copyfrom
INSERT INTO broadcast_recipients (broadcast_id, contact_id, status) VALUES ($1, $2, $3);

-- name: ListBroadcastRecipients :many
SELECT r.id, r.broadcast_id, r.contact_id, r.status, r.sent_at, r.error, r.created_at,
       c.name as contact_name, cp.phone as contact_phone
FROM broadcast_recipients r
JOIN contacts c ON r.contact_id = c.id
LEFT JOIN contact_phones cp ON cp.contact_id = c.id AND cp.label = 'primary'
WHERE r.broadcast_id = $1
ORDER BY r.created_at ASC LIMIT $2 OFFSET $3;

-- name: CountBroadcastRecipients :one
SELECT COUNT(*) FROM broadcast_recipients WHERE broadcast_id = $1;

-- name: CountBroadcastRecipientsByStatus :one
SELECT COUNT(*) FROM broadcast_recipients WHERE broadcast_id = $1 AND status = $2;

-- name: UpdateBroadcastRecipientStatus :exec
UPDATE broadcast_recipients SET status = $1, sent_at = $2, error = $3 WHERE id = $4;

-- name: GetPendingRecipientsForBroadcast :many
SELECT r.id, r.broadcast_id, r.contact_id, r.status, r.sent_at, r.error, r.created_at,
       cp.phone as contact_phone
FROM broadcast_recipients r
JOIN contacts c ON r.contact_id = c.id
LEFT JOIN contact_phones cp ON cp.contact_id = c.id AND cp.label = 'primary'
WHERE r.broadcast_id = $1 AND r.status = 'pending'
ORDER BY r.created_at ASC
LIMIT $2;

-- name: FindContactsByTag :many
SELECT c.id, c.tenant_id, c.name, c.email, c.language, c.created_at, c.updated_at
FROM contacts c
JOIN contact_tag_links l ON c.id = l.contact_id
JOIN contact_tags t ON l.tag_id = t.id
WHERE c.tenant_id = $1 AND t.name = $2;
