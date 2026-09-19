-- Add the supported Chinese providers to existing Channel Monitor V2 configs.
-- Append only missing entries so operator-defined enablement, model lists,
-- custom metadata, and platform ordering remain untouched.
WITH required_platforms (platform, ordinal) AS (
    VALUES
        ('kimi', 1),
        ('zhipu', 2),
        ('deepseek', 3)
)
UPDATE channel_monitor_v2_config AS config
SET
    platforms = config.platforms || (
        SELECT jsonb_agg(
            jsonb_build_object(
                'platform', required.platform,
                'enabled', TRUE,
                'models', '[]'::jsonb
            )
            ORDER BY required.ordinal
        )
        FROM required_platforms AS required
        WHERE NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(config.platforms) AS existing
            WHERE LOWER(TRIM(existing ->> 'platform')) = required.platform
        )
    ),
    version = config.version + 1,
    updated_at = NOW()
WHERE config.id = 1
  AND jsonb_typeof(config.platforms) = 'array'
  AND EXISTS (
      SELECT 1
      FROM required_platforms AS required
      WHERE NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(config.platforms) AS existing
          WHERE LOWER(TRIM(existing ->> 'platform')) = required.platform
      )
  );
