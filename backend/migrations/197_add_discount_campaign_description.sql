ALTER TABLE discount_campaigns
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

