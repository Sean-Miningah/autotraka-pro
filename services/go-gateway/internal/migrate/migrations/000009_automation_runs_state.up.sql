-- Add execution state tracking to automation_runs
ALTER TYPE automation_run_status ADD VALUE 'paused';
ALTER TYPE automation_run_status ADD VALUE 'waiting';

ALTER TABLE automation_runs
    ADD COLUMN current_node_id TEXT,
    ADD COLUMN variables JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN resume_at TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX idx_automation_runs_status ON automation_runs(status);
CREATE INDEX idx_automation_runs_resume_at ON automation_runs(resume_at) WHERE resume_at IS NOT NULL;