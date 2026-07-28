-- Durable welfare and compensation grant batches.
-- Amounts are kept separate from recharge records so grants never affect
-- total_recharged or affiliate rebate accounting.

CREATE TABLE IF NOT EXISTS benefit_grant_batches (
    id BIGSERIAL PRIMARY KEY,
    grant_type VARCHAR(20) NOT NULL CHECK (grant_type IN ('welfare', 'compensation')),
    grant_mode VARCHAR(24) NOT NULL CHECK (grant_mode IN ('fixed', 'percentage_24h')),
    audience_type VARCHAR(20) NOT NULL CHECK (audience_type IN ('all', 'selected')),
    fixed_amount NUMERIC(20,8),
    percentage NUMERIC(11,8),
    min_amount NUMERIC(20,8),
    per_user_cap NUMERIC(20,8),
    total_budget_cap NUMERIC(20,8),
    reason TEXT NOT NULL,
    notification_title VARCHAR(200) NOT NULL,
    notification_content TEXT NOT NULL,
    window_start TIMESTAMPTZ,
    window_end TIMESTAMPTZ,
    status VARCHAR(24) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'pending', 'processing', 'completed', 'partially_failed', 'failed', 'expired')),
    eligible_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    total_base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    total_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    distributed_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    average_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    max_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    executed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS benefit_grant_items (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES benefit_grant_batches(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    user_email_snapshot VARCHAR(255) NOT NULL,
    username_snapshot VARCHAR(100) NOT NULL DEFAULT '',
    base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    amount NUMERIC(20,8) NOT NULL CHECK (amount > 0),
    balance_before NUMERIC(20,8),
    balance_after NUMERIC(20,8),
    status VARCHAR(24) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'succeeded', 'failed', 'skipped_ineligible')),
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT benefit_grant_items_batch_user_unique UNIQUE (batch_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_batches_status
    ON benefit_grant_batches(status, created_at);
CREATE INDEX IF NOT EXISTS idx_benefit_grant_batches_created_by
    ON benefit_grant_batches(created_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_benefit_grant_items_batch_status
    ON benefit_grant_items(batch_id, status, id);
CREATE INDEX IF NOT EXISTS idx_benefit_grant_items_user_created
    ON benefit_grant_items(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_benefit_grant_items_user_unread
    ON benefit_grant_items(user_id, read_at, created_at DESC)
    WHERE status = 'succeeded';
