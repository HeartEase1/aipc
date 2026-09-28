-- Support user history queries without scanning all campaigns' claims.
CREATE INDEX IF NOT EXISTS idx_recharge_bonus_claims_user_credited
ON recharge_bonus_claims (user_id, credited_at DESC, order_id DESC)
WHERE credited_at IS NOT NULL;
