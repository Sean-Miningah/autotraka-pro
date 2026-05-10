-- name: CreateContact :one
INSERT INTO contacts (tenant_id, name, email, language) VALUES ($1, $2, $3, $4) RETURNING id, tenant_id, name, email, language, created_at, updated_at;

-- name: GetContactByID :one
SELECT id, tenant_id, name, email, language, created_at, updated_at FROM contacts WHERE id = $1 AND tenant_id = $2;

-- name: ListContactsByTenant :many
SELECT id, tenant_id, name, email, language, created_at, updated_at FROM contacts WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateContact :one
UPDATE contacts SET name = $1, email = $2, language = $3, updated_at = now() WHERE id = $4 AND tenant_id = $5 RETURNING id, tenant_id, name, email, language, created_at, updated_at;

-- name: DeleteContact :exec
DELETE FROM contacts WHERE id = $1 AND tenant_id = $2;

-- name: CountContactsByTenant :one
SELECT COUNT(*) FROM contacts WHERE tenant_id = $1;

-- name: CreateContactPhone :one
INSERT INTO contact_phones (contact_id, phone, label) VALUES ($1, $2, $3) ON CONFLICT (contact_id, phone) DO NOTHING RETURNING id, contact_id, phone, label, created_at;

-- name: ListContactPhones :many
SELECT id, contact_id, phone, label, created_at FROM contact_phones WHERE contact_id = $1;

-- name: DeleteContactPhonesByContactID :exec
DELETE FROM contact_phones WHERE contact_id = $1;

-- name: GetContactByPhone :one
SELECT c.id, c.tenant_id, c.name, c.email, c.language, c.created_at, c.updated_at
FROM contacts c
JOIN contact_phones cp ON c.id = cp.contact_id
WHERE c.tenant_id = $1 AND cp.phone = $2;

-- name: CreateChannelIdentity :one
INSERT INTO channel_identities (contact_id, channel_type, channel_identity) VALUES ($1, $2, $3) ON CONFLICT (contact_id, channel_type, channel_identity) DO NOTHING RETURNING id, contact_id, channel_type, channel_identity, created_at;

-- name: ListChannelIdentities :many
SELECT id, contact_id, channel_type, channel_identity, created_at FROM channel_identities WHERE contact_id = $1;

-- name: DeleteChannelIdentitiesByContactID :exec
DELETE FROM channel_identities WHERE contact_id = $1;

-- name: GetContactByChannelIdentity :one
SELECT c.id, c.tenant_id, c.name, c.email, c.language, c.created_at, c.updated_at
FROM contacts c
JOIN channel_identities ci ON c.id = ci.contact_id
WHERE c.tenant_id = $1 AND ci.channel_type = $2 AND ci.channel_identity = $3;

-- name: MoveContactPhones :exec
UPDATE contact_phones SET contact_id = $1 WHERE contact_id = $2;

-- name: MoveChannelIdentities :exec
UPDATE channel_identities SET contact_id = $1 WHERE contact_id = $2;
