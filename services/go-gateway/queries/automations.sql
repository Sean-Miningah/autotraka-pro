-- name: CreateAutomation :one
INSERT INTO automations (tenant_id, name, status, definition)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, name, status, definition, created_at, updated_at;

-- name: GetAutomationByID :one
SELECT id, tenant_id, name, status, definition, created_at, updated_at
FROM automations WHERE id = $1 AND tenant_id = $2;

-- name: ListAutomationsByTenant :many
SELECT id, tenant_id, name, status, definition, created_at, updated_at
FROM automations WHERE tenant_id = $1
ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountAutomationsByTenant :one
SELECT COUNT(*) FROM automations WHERE tenant_id = $1;

-- name: ListActiveAutomationsByTenant :many
SELECT id, tenant_id, name, status, definition, created_at, updated_at
FROM automations WHERE tenant_id = $1 AND status = 'active';

-- name: UpdateAutomation :one
UPDATE automations SET
    name = $1,
    status = $2,
    definition = $3,
    updated_at = now()
WHERE id = $4 AND tenant_id = $5
RETURNING id, tenant_id, name, status, definition, created_at, updated_at;

-- name: DeleteAutomation :exec
DELETE FROM automations WHERE id = $1 AND tenant_id = $2;

-- name: CreateAutomationRun :one
INSERT INTO automation_runs (automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at;

-- name: GetAutomationRunByID :one
SELECT id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at
FROM automation_runs WHERE id = $1;

-- name: ListAutomationRunsByAutomation :many
SELECT id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at
FROM automation_runs WHERE automation_id = $1
ORDER BY started_at DESC;

-- name: UpdateAutomationRunStatus :one
UPDATE automation_runs SET status = $1, completed_at = now(), updated_at = now()
WHERE id = $2
RETURNING id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at;

-- name: UpdateAutomationRunState :one
UPDATE automation_runs SET
    status = $1,
    current_node_id = $2,
    variables = $3,
    resume_at = $4,
    updated_at = now()
WHERE id = $5
RETURNING id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at;

-- name: PollAutomationRunsForResume :many
SELECT id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at
FROM automation_runs
WHERE status IN ('paused', 'waiting') AND resume_at <= now()
ORDER BY resume_at ASC
FOR UPDATE SKIP LOCKED
LIMIT $1;

-- name: GetWaitingRunsByConversation :many
SELECT id, automation_id, tenant_id, conversation_id, trigger_message_id, status, current_node_id, variables, resume_at, started_at, completed_at, updated_at
FROM automation_runs
WHERE conversation_id = $1 AND status = 'waiting';
