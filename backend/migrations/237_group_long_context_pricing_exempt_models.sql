ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS long_context_pricing_exempt_models JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE groups
SET long_context_pricing_exempt_models = '[]'::jsonb
WHERE long_context_pricing_exempt_models IS NULL;
