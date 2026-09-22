-- Bonus amounts are separate from paid principal and price discounts.
CREATE TABLE recharge_bonus_posters (
 id BIGSERIAL PRIMARY KEY, object_key TEXT NOT NULL UNIQUE, storage_config TEXT NOT NULL,
 size_bytes BIGINT NOT NULL CHECK(size_bytes > 0), state TEXT NOT NULL DEFAULT 'uploading',
 delete_after TIMESTAMPTZ NOT NULL DEFAULT NOW()+INTERVAL '24 hours',
 failures INT NOT NULL DEFAULT 0, last_error TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ,
 CHECK(state IN ('uploading','ready','deleting','deleted'))
);
CREATE TABLE recharge_bonus_campaigns (
 id BIGSERIAL PRIMARY KEY, title VARCHAR(160) NOT NULL, copy TEXT NOT NULL DEFAULT '',
 poster_id BIGINT REFERENCES recharge_bonus_posters(id),
 settlement_currency VARCHAR(8) NOT NULL DEFAULT 'CNY', enabled BOOLEAN NOT NULL DEFAULT FALSE, ever_enabled BOOLEAN NOT NULL DEFAULT FALSE,
 frequency VARCHAR(24) NOT NULL DEFAULT 'every_payment',
 starts_at TIMESTAMPTZ NOT NULL, ends_at TIMESTAMPTZ NOT NULL,
 timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
 created_by BIGINT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 deleted_at TIMESTAMPTZ, CHECK(frequency IN ('every_payment','daily_first','campaign_first')), CHECK(ends_at > starts_at)
);
CREATE TABLE recharge_bonus_tiers (
 id BIGSERIAL PRIMARY KEY, campaign_id BIGINT NOT NULL REFERENCES recharge_bonus_campaigns(id),
 min_amount NUMERIC(20,8) NOT NULL, max_amount NUMERIC(20,8), bonus_percent NUMERIC(8,4) NOT NULL,
 CHECK(min_amount >= 0 AND (max_amount IS NULL OR max_amount > min_amount)), CHECK(bonus_percent > 0 AND bonus_percent <= 100)
);
CREATE INDEX idx_recharge_bonus_active ON recharge_bonus_campaigns(settlement_currency,starts_at,ends_at) WHERE enabled AND deleted_at IS NULL;
CREATE INDEX idx_recharge_bonus_tiers ON recharge_bonus_tiers(campaign_id,min_amount);
CREATE TABLE recharge_bonus_claims (
 order_id BIGINT PRIMARY KEY REFERENCES payment_orders(id), campaign_id BIGINT NOT NULL REFERENCES recharge_bonus_campaigns(id),
 user_id BIGINT NOT NULL, period_key TEXT NOT NULL, bonus_amount NUMERIC(20,8) NOT NULL CHECK(bonus_amount>0),
 credited_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(campaign_id,user_id,period_key)
);
CREATE TABLE recharge_bonus_impressions (
 user_id BIGINT NOT NULL, local_date DATE NOT NULL, PRIMARY KEY(user_id,local_date)
);
CREATE INDEX idx_recharge_bonus_posters_cleanup ON recharge_bonus_posters(delete_after) WHERE state <> 'deleted';
-- Immutable pricing_snapshot.bonus holds rules, eligibility decision, credit and refund results.
