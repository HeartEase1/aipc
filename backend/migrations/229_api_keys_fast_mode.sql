ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS fast_mode BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN api_keys.fast_mode IS
    'Force OpenAI requests authenticated by this key to use priority service tier';
