-- Add runner type and config to agents table
ALTER TABLE agents ADD COLUMN runner_type TEXT NOT NULL DEFAULT 'claude';
ALTER TABLE agents ADD COLUMN runner_config JSONB;
