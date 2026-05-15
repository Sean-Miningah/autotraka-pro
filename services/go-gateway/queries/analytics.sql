-- name: UpsertAnalyticsDaily :exec
INSERT INTO analytics_daily (tenant_id, date, channel_type, metric_type, value)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, date, channel_type, metric_type)
DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: GetAnalyticsDailyByTenantAndDateRange :many
SELECT id, tenant_id, date, channel_type, metric_type, value, created_at, updated_at
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3
ORDER BY date ASC, metric_type ASC;

-- name: GetAnalyticsDailyByTenantDateRangeAndMetric :many
SELECT id, tenant_id, date, channel_type, metric_type, value, created_at, updated_at
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3 AND metric_type = $4
ORDER BY date ASC;

-- name: GetAnalyticsDailyByTenantDateRangeAndChannel :many
SELECT id, tenant_id, date, channel_type, metric_type, value, created_at, updated_at
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3 AND channel_type = $4
ORDER BY date ASC, metric_type ASC;

-- name: GetAnalyticsOverview :many
SELECT metric_type, SUM(value)::double precision AS total_value
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3
GROUP BY metric_type
ORDER BY metric_type;

-- name: GetAnalyticsConversationsByStatus :many
SELECT metric_type, channel_type, SUM(value)::double precision AS total_value
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3 AND metric_type LIKE 'conversations_%'
GROUP BY metric_type, channel_type
ORDER BY metric_type, channel_type;

-- name: GetMessageVolumeByDate :many
SELECT date, channel_type, metric_type, value
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3 AND metric_type IN ('messages_inbound', 'messages_outbound')
ORDER BY date ASC, channel_type ASC;

-- name: GetMessageVolumeCursor :many
SELECT id, tenant_id, date, channel_type, metric_type, value, created_at, updated_at
FROM analytics_daily
WHERE tenant_id = $1 AND date >= $2 AND date <= $3 AND metric_type IN ('messages_inbound', 'messages_outbound')
AND id > $4
ORDER BY id ASC LIMIT $5;

-- name: AggregateConversationsByStatus :many
SELECT status, COUNT(*) AS count
FROM conversations
WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY status;

-- name: AggregateConversationsByChannel :many
SELECT c.channel_type, cv.status, COUNT(DISTINCT cv.id) AS count
FROM conversations cv
JOIN messages m ON m.conversation_id = cv.id AND m.tenant_id = cv.tenant_id
JOIN channels c ON c.id = m.channel_id
WHERE cv.tenant_id = $1 AND cv.created_at >= $2 AND cv.created_at < $3
GROUP BY c.channel_type, cv.status;

-- name: AggregateConversationsTotalByChannel :many
SELECT c.channel_type, COUNT(DISTINCT cv.id) AS count
FROM conversations cv
JOIN messages m ON m.conversation_id = cv.id AND m.tenant_id = cv.tenant_id
JOIN channels c ON c.id = m.channel_id
WHERE cv.tenant_id = $1 AND cv.created_at >= $2 AND cv.created_at < $3
GROUP BY c.channel_type;

-- name: AggregateConversationsByHandledBy :many
SELECT handled_by, COUNT(*) AS count
FROM conversations
WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY handled_by;

-- name: AggregateConversationsByAgent :many
SELECT assigned_member_id, COUNT(*) AS count
FROM conversations
WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY assigned_member_id;

-- name: AggregateMessagesByDirection :many
SELECT direction, COUNT(*) AS count
FROM messages
WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY direction;

-- name: AggregateMessagesByChannel :many
SELECT direction, c.channel_type, COUNT(*) AS count
FROM messages m
JOIN channels c ON c.id = m.channel_id
WHERE m.tenant_id = $1 AND m.created_at >= $2 AND m.created_at < $3
GROUP BY m.direction, c.channel_type;

-- name: AggregateMessagesTotalByChannel :many
SELECT c.channel_type, COUNT(*) AS count
FROM messages m
JOIN channels c ON c.id = m.channel_id
WHERE m.tenant_id = $1 AND m.created_at >= $2 AND m.created_at < $3
GROUP BY c.channel_type;

-- name: ListTenants :many
SELECT id, name, mode, created_at, updated_at FROM tenants ORDER BY id;