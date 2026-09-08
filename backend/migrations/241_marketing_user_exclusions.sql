-- Per-user exclusions for server-side marketing rules.
-- Scopes: usage, recharge, membership, or all.
CREATE TABLE IF NOT EXISTS marketing_user_exclusions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT marketing_user_exclusions_scope_valid CHECK (scope IN ('usage','recharge','membership','all')),
    UNIQUE (user_id, scope)
);
CREATE INDEX IF NOT EXISTS idx_marketing_user_exclusions_scope_user
    ON marketing_user_exclusions (scope, user_id) WHERE enabled = TRUE;
