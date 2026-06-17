-- +goose Up
ALTER TABLE agents ADD COLUMN mode TEXT NOT NULL DEFAULT 'worktree'
    CHECK (mode IN ('worktree', 'bare'));
CREATE INDEX IF NOT EXISTS idx_agents_mode ON agents(mode);

-- +goose Down
DROP INDEX IF EXISTS idx_agents_mode;
ALTER TABLE agents DROP COLUMN mode;
