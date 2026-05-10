-- name: CreateTenant :one
INSERT INTO tenants (name, mode) VALUES ($1, $2) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: CreateMember :one
INSERT INTO members (tenant_id, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetMemberByEmail :one
SELECT * FROM members WHERE tenant_id = $1 AND email = $2;

-- name: GetMemberByID :one
SELECT * FROM members WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (member_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE id = $1;

-- name: DeleteRefreshTokensByMember :exec
DELETE FROM refresh_tokens WHERE member_id = $1;