CREATE TYPE automation_status AS ENUM ('draft', 'active');
CREATE TYPE automation_run_status AS ENUM ('running', 'completed', 'failed');

CREATE TABLE automations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status automation_status NOT NULL DEFAULT 'draft',
    definition JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_automations_tenant_id ON automations(tenant_id);
CREATE INDEX idx_automations_status ON automations(status);
CREATE INDEX idx_automations_tenant_status ON automations(tenant_id, status);

CREATE TABLE automation_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    automation_id UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    trigger_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    status automation_run_status NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_automation_runs_automation_id ON automation_runs(automation_id);
CREATE INDEX idx_automation_runs_tenant_id ON automation_runs(tenant_id);
CREATE INDEX idx_automation_runs_conversation_id ON automation_runs(conversation_id);
