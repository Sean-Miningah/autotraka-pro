-- name: CreateTemplate :one
INSERT INTO templates (tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at;

-- name: GetTemplateByID :one
SELECT id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at
FROM templates WHERE id = $1 AND tenant_id = $2;

-- name: ListTemplatesByTenant :many
SELECT id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at
FROM templates WHERE tenant_id = $1
ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountTemplatesByTenant :one
SELECT COUNT(*) FROM templates WHERE tenant_id = $1;

-- name: ListTemplatesByTenantAndStatus :many
SELECT id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at
FROM templates WHERE tenant_id = $1 AND status = $2
ORDER BY updated_at DESC;

-- name: UpdateTemplate :one
UPDATE templates SET
    name = $1,
    category = $2,
    status = $3,
    language = $4,
    body = $5,
    parameters = $6,
    meta_template_id = $7,
    updated_at = now()
WHERE id = $8 AND tenant_id = $9
RETURNING id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at;

-- name: ListPendingTemplates :many
SELECT id, tenant_id, channel_id, parent_template_id, name, category, status, language, body, parameters, meta_template_id, created_at, updated_at
FROM templates WHERE status = 'pending';

-- name: DeleteTemplate :exec
DELETE FROM templates WHERE id = $1 AND tenant_id = $2;
