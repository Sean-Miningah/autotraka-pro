DROP TABLE IF EXISTS scheduler_locks;

ALTER TABLE channels DROP COLUMN IF EXISTS health_status;
ALTER TABLE channels DROP COLUMN IF EXISTS health_checked_at;
ALTER TABLE channels DROP COLUMN IF EXISTS last_error;
