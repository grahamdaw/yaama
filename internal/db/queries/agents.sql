-- name: ListAgents :many
SELECT id, name, tmux_session, status, task, branch, created_at, updated_at
FROM agents
ORDER BY id ASC;
