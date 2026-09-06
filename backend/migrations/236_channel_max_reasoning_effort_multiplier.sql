ALTER TABLE channel_model_pricing
    ADD COLUMN IF NOT EXISTS max_reasoning_effort_multiplier NUMERIC(20, 10);

ALTER TABLE channel_model_pricing
    ADD CONSTRAINT channel_model_pricing_max_reasoning_effort_multiplier_positive
    CHECK (max_reasoning_effort_multiplier > 0
       AND max_reasoning_effort_multiplier < 'Infinity'::numeric);
