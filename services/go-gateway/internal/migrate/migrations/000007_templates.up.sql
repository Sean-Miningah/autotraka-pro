CREATE TYPE template_status AS ENUM ('draft', 'pending', 'approved', 'rejected');

CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    parent_template_id UUID REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    status template_status NOT NULL DEFAULT 'draft',
    language TEXT NOT NULL DEFAULT 'en',
    body TEXT NOT NULL,
    parameters JSONB NOT NULL DEFAULT '[]',
    meta_template_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_templates_tenant_id ON templates(tenant_id);
CREATE INDEX idx_templates_status ON templates(status);
CREATE INDEX idx_templates_parent_template_id ON templates(parent_template_id);
CREATE INDEX idx_templates_tenant_status ON templates(tenant_id, status);
