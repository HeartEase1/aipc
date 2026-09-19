-- Preserve the exact payment amount for currencies with three fractional
-- digits (for example KWD/BHD). Existing values are unchanged by widening
-- the numeric scale. refund_amount is a platform-balance amount, not a
-- settlement-currency amount, so it intentionally keeps its existing scale.
ALTER TABLE payment_orders
    ALTER COLUMN pay_amount TYPE DECIMAL(20,8);

-- Keep the persisted settlement currency aligned with the immutable provider
-- snapshot when the snapshot contains a valid currency. This is idempotent
-- and also repairs deployments that applied the previous backfill partially.
UPDATE payment_orders
SET settlement_currency = UPPER(BTRIM(provider_snapshot->>'currency'))
WHERE order_type = 'balance'
  AND pricing_snapshot IS NULL
  AND UPPER(BTRIM(provider_snapshot->>'currency')) ~ '^[A-Z]{3}$'
  AND settlement_currency IS DISTINCT FROM UPPER(BTRIM(provider_snapshot->>'currency'));
