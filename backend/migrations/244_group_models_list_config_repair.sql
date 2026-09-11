-- 244: 收敛 groups.models_list_config，兼容从官方库切过来或手工回滚后缺列的实例。
--
-- AIPC 使用 models_list_config，不能把该列重命名成官方的 model_allowlist，
-- 否则密钥页/订阅页会因为缺列 500。本迁移可重放：
--   1) 只有官方 model_allowlist、没有 models_list_config -> 重命名过来，数据保留；
--   2) 两列并存 -> models_list_config 仍是默认空值时，回填官方列配置；
--   3) 两列都没有 -> 按默认值补建 models_list_config。
-- 结束后保证 groups.models_list_config 一定存在，且为 NOT NULL DEFAULT '{}'.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'groups'::regclass
          AND attname = 'model_allowlist'
          AND NOT attisdropped
    ) THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_attribute
            WHERE attrelid = 'groups'::regclass
              AND attname = 'models_list_config'
              AND NOT attisdropped
        ) THEN
            ALTER TABLE groups RENAME COLUMN model_allowlist TO models_list_config;
        ELSE
            EXECUTE $backfill$
                UPDATE groups
                   SET models_list_config = model_allowlist
                 WHERE COALESCE(models_list_config, '{}'::jsonb) = '{}'::jsonb
                   AND COALESCE(model_allowlist, '{}'::jsonb) <> '{}'::jsonb
            $backfill$;
        END IF;
    END IF;
END
$$;

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS models_list_config JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE groups SET models_list_config = '{}'::jsonb WHERE models_list_config IS NULL;

ALTER TABLE groups ALTER COLUMN models_list_config SET DEFAULT '{}'::jsonb;
ALTER TABLE groups ALTER COLUMN models_list_config SET NOT NULL;

COMMENT ON COLUMN groups.models_list_config IS
    'Group model list config: constrains both model listing responses and request admission';
