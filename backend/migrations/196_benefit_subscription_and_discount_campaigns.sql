-- Subscription-aware benefit grants and scheduled token discount campaigns.

ALTER TABLE benefit_grant_batches
    ADD COLUMN IF NOT EXISTS include_subscription BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS subscription_percentage NUMERIC(11,8),
    ADD COLUMN IF NOT EXISTS total_balance_base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_subscription_base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_balance_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_subscription_amount NUMERIC(20,8) NOT NULL DEFAULT 0;

ALTER TABLE benefit_grant_items
    ADD COLUMN IF NOT EXISTS balance_base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subscription_base_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subscription_amount NUMERIC(20,8) NOT NULL DEFAULT 0;

-- Preserve the component view for batches created before this migration.
UPDATE benefit_grant_batches
SET total_balance_base_cost = total_base_cost,
    total_balance_amount = total_amount
WHERE total_balance_base_cost = 0
  AND total_subscription_base_cost = 0
  AND total_subscription_amount = 0
  AND (total_base_cost <> 0 OR total_amount <> 0);

UPDATE benefit_grant_items
SET balance_base_cost = base_cost,
    balance_amount = amount
WHERE balance_base_cost = 0
  AND subscription_base_cost = 0
  AND subscription_amount = 0
  AND (base_cost <> 0 OR amount <> 0);

CREATE TABLE IF NOT EXISTS discount_campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    schedule_type VARCHAR(16) NOT NULL
        CHECK (schedule_type IN ('one_time', 'weekly')),
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    weekdays SMALLINT[] NOT NULL DEFAULT '{}',
    start_minute SMALLINT,
    end_minute SMALLINT,
    all_day BOOLEAN NOT NULL DEFAULT FALSE,
    discount_factor NUMERIC(8,6) NOT NULL
        CHECK (discount_factor > 0 AND discount_factor <= 1),
    min_effective_multiplier NUMERIC(10,6),
    budget_cap NUMERIC(20,8),
    discount_spent NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT discount_campaign_one_time_window CHECK (
        schedule_type <> 'one_time'
        OR (starts_at IS NOT NULL AND ends_at IS NOT NULL AND starts_at < ends_at)
    ),
    CONSTRAINT discount_campaign_weekly_window CHECK (
        schedule_type <> 'weekly'
        OR (
            cardinality(weekdays) > 0
            AND (all_day OR (start_minute BETWEEN 0 AND 1439 AND end_minute BETWEEN 0 AND 1439 AND start_minute <> end_minute))
        )
    ),
    CONSTRAINT discount_campaign_min_multiplier CHECK (
        min_effective_multiplier IS NULL OR min_effective_multiplier > 0
    ),
    CONSTRAINT discount_campaign_budget CHECK (budget_cap IS NULL OR budget_cap > 0)
);

CREATE INDEX IF NOT EXISTS idx_discount_campaigns_runtime
    ON discount_campaigns(enabled, deleted_at, starts_at, ends_at);

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS discount_campaign_id BIGINT REFERENCES discount_campaigns(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS discount_factor NUMERIC(8,6),
    ADD COLUMN IF NOT EXISTS original_rate_multiplier NUMERIC(10,6),
    ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(20,10) NOT NULL DEFAULT 0;

-- Ent may create the scalar column before SQL migrations run. Add the foreign
-- key independently so IF NOT EXISTS on the column cannot skip referential integrity.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'usage_logs_discount_campaign_id_fkey'
          AND conrelid = 'usage_logs'::regclass
    ) THEN
        ALTER TABLE usage_logs
            ADD CONSTRAINT usage_logs_discount_campaign_id_fkey
            FOREIGN KEY (discount_campaign_id)
            REFERENCES discount_campaigns(id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_usage_logs_discount_campaign
    ON usage_logs(discount_campaign_id, created_at DESC)
    WHERE discount_campaign_id IS NOT NULL;
