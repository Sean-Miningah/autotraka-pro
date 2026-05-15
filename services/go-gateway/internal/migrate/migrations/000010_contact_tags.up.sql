CREATE TABLE contact_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_contact_tags_tenant_id ON contact_tags(tenant_id);

CREATE TABLE contact_tag_links (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES contact_tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (contact_id, tag_id)
);

CREATE INDEX idx_contact_tag_links_contact_id ON contact_tag_links(contact_id);
CREATE INDEX idx_contact_tag_links_tag_id ON contact_tag_links(tag_id);
