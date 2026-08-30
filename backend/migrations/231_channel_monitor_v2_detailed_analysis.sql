-- User-facing Channel Monitor V2 defaults to the concise status-card view.
-- Administrators can opt in to the heavier matrix/trend/model/error analysis.
INSERT INTO settings (key, value)
VALUES ('channel_monitor_v2_detailed_analysis_enabled', 'false')
ON CONFLICT (key) DO NOTHING;
