CREATE TABLE scheduler_locks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_name TEXT NOT NULL UNIQUE,
    locked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_by TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '5 minutes')
);

CREATE INDEX idx_scheduler_locks_task_name ON scheduler_locks(task_name);

ALTER TABLE channels ADD COLUMN health_status TEXT DEFAULT 'unknown';
ALTER TABLE channels ADD COLUMN health_checked_at TIMESTAMPTZ;
ALTER TABLE channels ADD COLUMN last_error TEXT;
