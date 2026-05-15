DROP INDEX IF EXISTS idx_automation_runs_resume_at;
DROP INDEX IF EXISTS idx_automation_runs_status;

ALTER TABLE automation_runs
    DROP COLUMN IF EXISTS current_node_id,
    DROP COLUMN IF EXISTS variables,
    DROP COLUMN IF EXISTS resume_at,
    DROP COLUMN IF EXISTS updated_at;

-- Note: PostgreSQL does not support removing enum values.
-- The 'paused' and 'waiting' values remain in the type but are unused.