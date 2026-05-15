CREATE TYPE broadcast_status AS ENUM ('draft', 'scheduled', 'sending', 'completed', 'failed');
CREATE TYPE broadcast_recipient_status AS ENUM ('pending', 'sent', 'failed');

CREATE TABLE broadcasts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    template_id UUID REFERENCES templates(id) ON DELETE SET NULL,
    parameters JSONB NOT NULL DEFAULT '{}',
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    status broadcast_status NOT NULL DEFAULT 'draft',
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_broadcasts_tenant_id ON broadcasts(tenant_id);
CREATE INDEX idx_broadcasts_status ON broadcasts(status);
CREATE INDEX idx_broadcasts_scheduled_at ON broadcasts(scheduled_at) WHERE scheduled_at IS NOT NULL;
CREATE INDEX idx_broadcasts_tenant_status ON broadcasts(tenant_id, status);

CREATE TABLE broadcast_recipients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    broadcast_id UUID NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    status broadcast_recipient_status NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(broadcast_id, contact_id)
);

CREATE INDEX idx_broadcast_recipients_broadcast_id ON broadcast_recipients(broadcast_id);
CREATE INDEX idx_broadcast_recipients_contact_id ON broadcast_recipients(contact_id);
CREATE INDEX idx_broadcast_recipients_status ON broadcast_recipients(status);
