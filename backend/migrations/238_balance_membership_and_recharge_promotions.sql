-- Balance recharge membership and promotion foundation.
-- All promotion rows are disabled by default; existing orders keep their original pricing.

CREATE TABLE IF NOT EXISTS balance_membership_tiers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    settlement_currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    threshold_amount DECIMAL(20,8) NOT NULL,
    discount_percent DECIMAL(8,4) NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_membership_tiers_threshold_nonnegative CHECK (threshold_amount >= 0),
    CONSTRAINT balance_membership_tiers_discount_range CHECK (discount_percent >= 0 AND discount_percent < 100)
);
CREATE INDEX IF NOT EXISTS idx_balance_membership_tiers_lookup
    ON balance_membership_tiers (settlement_currency, enabled, threshold_amount DESC, sort_order, id);

CREATE TABLE IF NOT EXISTS recharge_promotions (
    id BIGSERIAL PRIMARY KEY,
    kind VARCHAR(20) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    settlement_currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    min_amount DECIMAL(20,8),
    max_amount DECIMAL(20,8),
    discount_percent DECIMAL(8,4) NOT NULL DEFAULT 0,
    max_discount_amount DECIMAL(20,8),
    budget_amount DECIMAL(20,8),
    reserved_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    redeemed_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recharge_promotions_kind_valid CHECK (kind IN ('recharge', 'first_recharge')),
    CONSTRAINT recharge_promotions_discount_range CHECK (discount_percent >= 0 AND discount_percent < 100),
    CONSTRAINT recharge_promotions_amount_range CHECK (min_amount IS NULL OR max_amount IS NULL OR min_amount <= max_amount),
    CONSTRAINT recharge_promotions_budget_nonnegative CHECK (budget_amount IS NULL OR budget_amount >= 0)
);
CREATE INDEX IF NOT EXISTS idx_recharge_promotions_active
    ON recharge_promotions (kind, settlement_currency, enabled, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS recharge_promotion_claims (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES recharge_promotions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    order_id BIGINT,
    source VARCHAR(20) NOT NULL,
    discount_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'reserved',
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    redeemed_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recharge_promotion_claims_source_valid CHECK (source IN ('first_recharge', 'campaign', 'membership')),
    CONSTRAINT recharge_promotion_claims_status_valid CHECK (status IN ('reserved', 'redeemed', 'released')),
    CONSTRAINT recharge_promotion_claims_discount_nonnegative CHECK (discount_amount >= 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_recharge_promotion_claims_first_recharge
    ON recharge_promotion_claims (user_id) WHERE source = 'first_recharge' AND status IN ('reserved', 'redeemed');
CREATE UNIQUE INDEX IF NOT EXISTS uq_recharge_promotion_claims_order
    ON recharge_promotion_claims (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_recharge_promotion_claims_user
    ON recharge_promotion_claims (user_id, created_at DESC);

ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS settlement_currency VARCHAR(8) NOT NULL DEFAULT 'CNY';
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS original_amount DECIMAL(20,8);
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS discounted_amount DECIMAL(20,8);
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS discount_source VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS pricing_snapshot JSONB;
CREATE INDEX IF NOT EXISTS idx_payment_orders_membership_window
    ON payment_orders (user_id, order_type, settlement_currency, status, paid_at);

UPDATE payment_orders
SET original_amount = COALESCE(original_amount, amount),
    discounted_amount = COALESCE(discounted_amount, amount)
WHERE original_amount IS NULL OR discounted_amount IS NULL;
