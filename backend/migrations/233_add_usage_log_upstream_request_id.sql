-- NULL for historical rows, WebSocket turns, or accounts without a configured header.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_request_id VARCHAR(128);
