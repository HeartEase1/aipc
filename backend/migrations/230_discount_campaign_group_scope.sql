ALTER TABLE discount_campaigns
    ADD COLUMN IF NOT EXISTS group_ids BIGINT[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN discount_campaigns.group_ids IS
    'Balance group IDs eligible for the campaign; an empty array means all balance groups';
