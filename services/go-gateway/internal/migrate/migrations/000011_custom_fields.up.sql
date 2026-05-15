CREATE TABLE custom_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN ('text', 'number', 'date', 'boolean', 'select')),
    options JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_custom_fields_tenant_id ON custom_fields(tenant_id);

CREATE TABLE contact_custom_fields (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_fields(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (contact_id, field_id)
);

CREATE INDEX idx_contact_custom_fields_contact_id ON contact_custom_fields(contact_id);
CREATE INDEX idx_contact_custom_fields_field_id ON contact_custom_fields(field_id);
