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

-- name: ListEnrichedConversationsByTenant :many
SELECT
    c.id, c.tenant_id, c.contact_id, c.status, c.assigned_member_id, c.handled_by,
    c.previous_conversation_id, c.created_at, c.updated_at,
    COALESCE(cont.name, '') AS contact_name,
    COALESCE(ci.channel_type, '') AS channel_type,
    COALESCE(last_msg.preview, '') AS last_message,
    COALESCE(last_msg.created_at, '1970-01-01 00:00:00'::timestamptz) AS last_message_at,
    COALESCE(cr.unread_count, 0) AS unread_count
FROM conversations c
LEFT JOIN contacts cont ON cont.id = c.contact_id
LEFT JOIN LATERAL (
    SELECT ci.channel_type
    FROM channel_identities ci
    WHERE ci.contact_id = c.contact_id
    ORDER BY ci.created_at ASC
    LIMIT 1
) ci ON true
LEFT JOIN LATERAL (
    SELECT m.content::text AS preview, m.created_at
    FROM messages m
    WHERE m.conversation_id = c.id
    ORDER BY m.created_at DESC
    LIMIT 1
) last_msg ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS unread_count
    FROM messages m
    WHERE m.conversation_id = c.id
    AND m.direction = 'inbound'
    AND m.created_at > COALESCE(
        (SELECT cr2.last_read_at FROM conversation_reads cr2 WHERE cr2.member_id = $2 AND cr2.conversation_id = c.id),
        '1970-01-01 00:00:00'::timestamptz
    )
) cr ON true
WHERE c.tenant_id = $1
ORDER BY c.updated_at DESC
LIMIT $3 OFFSET $4;

-- name: CountEnrichedConversationsByTenant :one
SELECT COUNT(*) FROM conversations WHERE tenant_id = $1;

-- name: UpsertConversationRead :one
INSERT INTO conversation_reads (member_id, conversation_id, last_read_at)
VALUES ($1, $2, now())
ON CONFLICT (member_id, conversation_id)
DO UPDATE SET last_read_at = now(), updated_at = now()
RETURNING id, member_id, conversation_id, last_read_at, created_at, updated_at;
