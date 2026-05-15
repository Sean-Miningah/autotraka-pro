CREATE TABLE analytics_daily (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    channel_type TEXT,
    metric_type TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(tenant_id, date, channel_type, metric_type)
);

CREATE INDEX idx_analytics_daily_tenant_date ON analytics_daily(tenant_id, date);
CREATE INDEX idx_analytics_daily_metric_type ON analytics_daily(tenant_id, date, metric_type);