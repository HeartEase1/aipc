-- Persist whether a subscription payment extends the current term or starts a new term.
ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS subscription_action VARCHAR(20) NOT NULL DEFAULT 'extend';

COMMENT ON COLUMN payment_orders.subscription_action IS
    'Subscription fulfillment action: extend keeps the current term, restart replaces it from payment completion time';

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_active_restart_unique
    ON payment_orders (user_id, subscription_group_id)
    WHERE subscription_action = 'restart'
      AND status IN ('PENDING', 'PAID', 'RECHARGING');
