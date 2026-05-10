-- name: CreateSchedulerLock :one
INSERT INTO scheduler_locks (task_name, locked_by, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (task_name) DO NOTHING
RETURNING id, task_name, locked_at, locked_by, expires_at;

-- name: GetSchedulerLock :one
SELECT id, task_name, locked_at, locked_by, expires_at FROM scheduler_locks WHERE task_name = $1;

-- name: UpdateSchedulerLock :one
UPDATE scheduler_locks SET locked_by = $2, locked_at = now(), expires_at = $3
WHERE task_name = $1
RETURNING id, task_name, locked_at, locked_by, expires_at;

-- name: DeleteSchedulerLock :exec
DELETE FROM scheduler_locks WHERE task_name = $1 AND locked_by = $2;

-- name: UpdateChannelHealth :exec
UPDATE channels SET health_status = $1, health_checked_at = now(), last_error = $2 WHERE id = $3;

-- name: GetChannelHealth :one
SELECT id, tenant_id, name, channel_type, health_status, health_checked_at, last_error FROM channels WHERE id = $1;

-- name: ListActiveChannels :many
SELECT id, tenant_id, name, channel_type, config, status, created_at, updated_at FROM channels WHERE status = 'active';
