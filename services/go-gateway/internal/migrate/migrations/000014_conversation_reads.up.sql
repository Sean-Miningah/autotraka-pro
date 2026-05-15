CREATE TABLE conversation_reads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(member_id, conversation_id)
);

CREATE INDEX idx_conversation_reads_member ON conversation_reads(member_id);
CREATE INDEX idx_conversation_reads_conversation ON conversation_reads(conversation_id);