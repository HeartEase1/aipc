-- Follow-up to 238. 238 is immutable after release; this migration adds the
-- fields needed to safely reconcile an unpaid provider order.
ALTER TABLE recharge_promotion_claims
    ADD COLUMN IF NOT EXISTS closed_confirmed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS checked_at TIMESTAMPTZ;

-- Claimed financial history must not disappear when an administrator deletes
-- a promotion. Existing deployments may have the 238 CASCADE foreign key.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'recharge_promotion_claims'::regclass
          AND conname = 'recharge_promotion_claims_promotion_id_fkey'
    ) THEN
        ALTER TABLE recharge_promotion_claims
            DROP CONSTRAINT recharge_promotion_claims_promotion_id_fkey;
    END IF;
    ALTER TABLE recharge_promotion_claims
        ADD CONSTRAINT recharge_promotion_claims_promotion_id_fkey
        FOREIGN KEY (promotion_id) REFERENCES recharge_promotions(id) ON DELETE RESTRICT;
END $$;

CREATE INDEX IF NOT EXISTS idx_recharge_promotion_claims_reconcile
    ON recharge_promotion_claims (status, checked_at, closed_confirmed_at, order_id);

-- Only recover a currency when the provider snapshot explicitly contains a
-- valid ISO code. Never infer historical face values from wallet `amount`.
UPDATE payment_orders
SET settlement_currency = UPPER(BTRIM(provider_snapshot->>'currency'))
WHERE pricing_snapshot IS NULL
  AND UPPER(BTRIM(provider_snapshot->>'currency')) ~ '^[A-Z]{3}$'
  AND settlement_currency IS DISTINCT FROM UPPER(BTRIM(provider_snapshot->>'currency'));
