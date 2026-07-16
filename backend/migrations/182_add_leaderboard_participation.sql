ALTER TABLE users
    ADD COLUMN IF NOT EXISTS leaderboard_enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS inviter_bound_at TIMESTAMPTZ;

-- Existing relationships do not retain their original bind timestamp. The
-- affiliate record creation time is the only safe historical baseline; new
-- bindings are recorded exactly by AffiliateRepository.BindInviter.
UPDATE user_affiliates
SET inviter_bound_at = created_at
WHERE inviter_id IS NOT NULL
  AND inviter_bound_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usage_logs_leaderboard_window
    ON usage_logs (created_at, user_id);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_leaderboard
    ON user_affiliate_ledger (action, created_at, user_id);

CREATE INDEX IF NOT EXISTS idx_user_affiliates_leaderboard_invites
    ON user_affiliates (inviter_id, inviter_bound_at)
    WHERE inviter_id IS NOT NULL;
