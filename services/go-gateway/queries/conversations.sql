-- name: CreateConversation :one
INSERT INTO conversations (tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at;

-- name: GetConversationByID :one
SELECT id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at
FROM conversations WHERE id = $1 AND tenant_id = $2;

-- name: ListConversationsByTenant :many
SELECT id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at
FROM conversations WHERE tenant_id = $1
ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountConversationsByTenant :one
SELECT COUNT(*) FROM conversations WHERE tenant_id = $1;

-- name: UpdateConversation :one
UPDATE conversations SET
    status = $1,
    assigned_member_id = $2,
    handled_by = $3,
    updated_at = now()
WHERE id = $4 AND tenant_id = $5
RETURNING id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at;

-- name: GetOpenConversationByContact :one
SELECT id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at
FROM conversations
WHERE tenant_id = $1 AND contact_id = $2 AND status != 'closed'
ORDER BY updated_at DESC LIMIT 1;

-- name: GetLastConversationByContact :one
SELECT id, tenant_id, contact_id, status, assigned_member_id, handled_by, previous_conversation_id, created_at, updated_at
FROM conversations
WHERE tenant_id = $1 AND contact_id = $2
ORDER BY updated_at DESC LIMIT 1;

-- name: CreateMessage :one
INSERT INTO messages (tenant_id, conversation_id, channel_id, direction, status, content_type, content)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, tenant_id, conversation_id, channel_id, direction, status, content_type, content, created_at, updated_at;

-- name: GetMessageByID :one
SELECT id, tenant_id, conversation_id, channel_id, direction, status, content_type, content, created_at, updated_at
FROM messages WHERE id = $1 AND tenant_id = $2;

-- name: ListMessagesByConversation :many
SELECT id, tenant_id, conversation_id, channel_id, direction, status, content_type, content, created_at, updated_at
FROM messages WHERE conversation_id = $1 AND tenant_id = $2
ORDER BY created_at ASC LIMIT $3 OFFSET $4;

-- name: CountMessagesByConversation :one
SELECT COUNT(*) FROM messages WHERE conversation_id = $1 AND tenant_id = $2;

-- name: UpdateMessageStatus :one
UPDATE messages SET status = $1, updated_at = now()
WHERE id = $2 AND tenant_id = $3
RETURNING id, tenant_id, conversation_id, channel_id, direction, status, content_type, content, created_at, updated_at;

-- name: GetLastMessageByConversation :one
SELECT id, tenant_id, conversation_id, channel_id, direction, status, content_type, content, created_at, updated_at
FROM messages WHERE conversation_id = $1 AND direction = 'inbound'
ORDER BY created_at DESC LIMIT 1;
