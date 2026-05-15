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

-- Tags
-- name: CreateContactTag :one
INSERT INTO contact_tags (tenant_id, name) VALUES ($1, $2) ON CONFLICT (tenant_id, name) DO NOTHING RETURNING id, tenant_id, name, created_at;

-- name: GetContactTagByName :one
SELECT id, tenant_id, name, created_at FROM contact_tags WHERE tenant_id = $1 AND name = $2;

-- name: AddTagToContact :exec
INSERT INTO contact_tag_links (contact_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveTagFromContact :exec
DELETE FROM contact_tag_links WHERE contact_id = $1 AND tag_id = $2;

-- name: ListContactTags :many
SELECT t.id, t.tenant_id, t.name, t.created_at
FROM contact_tags t
JOIN contact_tag_links l ON t.id = l.tag_id
WHERE l.contact_id = $1;

-- name: HasContactTag :one
SELECT EXISTS(SELECT 1 FROM contact_tag_links WHERE contact_id = $1 AND tag_id = $2);

-- Custom Fields
-- name: CreateCustomField :one
INSERT INTO custom_fields (tenant_id, name, field_type, options) VALUES ($1, $2, $3, $4) RETURNING id, tenant_id, name, field_type, options, created_at;

-- name: GetCustomFieldByName :one
SELECT id, tenant_id, name, field_type, options, created_at FROM custom_fields WHERE tenant_id = $1 AND name = $2;

-- name: SetContactCustomField :one
INSERT INTO contact_custom_fields (contact_id, field_id, value) VALUES ($1, $2, $3)
ON CONFLICT (contact_id, field_id) DO UPDATE SET value = $3, updated_at = now()
RETURNING contact_id, field_id, value, created_at, updated_at;

-- name: GetContactCustomField :one
SELECT contact_id, field_id, value, created_at, updated_at FROM contact_custom_fields WHERE contact_id = $1 AND field_id = $2;

-- name: ListContactCustomFields :many
SELECT f.id, f.tenant_id, f.name, f.field_type, f.options, v.value
FROM custom_fields f
JOIN contact_custom_fields v ON f.id = v.field_id
WHERE v.contact_id = $1;
