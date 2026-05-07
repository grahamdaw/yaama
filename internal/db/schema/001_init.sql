-- +goose Up
CREATE TABLE IF NOT EXISTS agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    tmux_session TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('idle', 'running', 'blocked', 'review', 'done')),
    task TEXT,
    last_activity TEXT,
    branch TEXT,
    working_dir TEXT,
    profile_name TEXT,
    ticket_id TEXT,
    initial_prompt TEXT,
    last_heartbeat_at DATETIME,
    last_error TEXT,
    cleanup_state TEXT NOT NULL DEFAULT 'active' CHECK (cleanup_state IN ('active', 'archived', 'pruned')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_cleanup_state ON agents(cleanup_state);
CREATE INDEX IF NOT EXISTS idx_agents_updated_at ON agents(updated_at);
CREATE INDEX IF NOT EXISTS idx_agents_ticket_id ON agents(ticket_id);

-- +goose Down
DROP TABLE IF EXISTS agents;
