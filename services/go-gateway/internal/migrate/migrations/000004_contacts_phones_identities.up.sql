CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT,
    email TEXT,
    language TEXT DEFAULT 'en',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contacts_tenant_id ON contacts(tenant_id);

CREATE TABLE contact_phones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    phone TEXT NOT NULL,
    label TEXT DEFAULT 'primary',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(contact_id, phone)
);

CREATE INDEX idx_contact_phones_contact_id ON contact_phones(contact_id);
CREATE INDEX idx_contact_phones_phone ON contact_phones(phone);

CREATE TABLE channel_identities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('whatsapp', 'instagram', 'facebook')),
    channel_identity TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(contact_id, channel_type, channel_identity)
);

CREATE INDEX idx_channel_identities_contact_id ON channel_identities(contact_id);
CREATE INDEX idx_channel_identities_lookup ON channel_identities(channel_type, channel_identity);
