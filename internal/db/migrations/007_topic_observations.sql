-- Topic-agent observation bindings
CREATE TABLE IF NOT EXISTS topic_agent_bindings (
    topic_id     BIGINT NOT NULL,
    agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    binding_type TEXT NOT NULL DEFAULT 'observe',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (topic_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_topic_agent_bindings_agent ON topic_agent_bindings(agent_id);
