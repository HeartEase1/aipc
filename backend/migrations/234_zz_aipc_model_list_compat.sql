-- Runs before official 235 renames models_list_config. AIPC's selection was
-- display-only; do not silently turn it into a request admission restriction.
-- Official installations with 235 already recorded retain official semantics.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM schema_migrations
        WHERE filename = '182_add_leaderboard_participation.sql'
    ) AND NOT EXISTS (
        SELECT 1 FROM schema_migrations
        WHERE filename = '235_group_model_allowlist.sql'
    ) AND EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'groups'::regclass
          AND attname = 'models_list_config' AND NOT attisdropped
    ) THEN
        UPDATE groups
           SET models_list_config = models_list_config || '{"legacy_list_only":true}'::jsonb
         WHERE jsonb_typeof(models_list_config) = 'object'
           AND models_list_config <> '{}'::jsonb;
    END IF;
END
$$;
