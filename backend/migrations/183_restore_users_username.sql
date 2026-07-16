-- Some legacy installations marked the original schema-parity migration as
-- applied before the optional profile field was present. Restore it here so
-- leaderboard queries can consistently use the user's public display name.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS username VARCHAR(100) NOT NULL DEFAULT '';
