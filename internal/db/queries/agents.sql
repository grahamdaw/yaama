-- name: ListAgents :many
SELECT
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at
FROM agents
ORDER BY id ASC;

-- name: ListActiveAgents :many
SELECT
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at
FROM agents
WHERE cleanup_state = 'active'
ORDER BY id ASC;

-- name: ListAgentsByStatus :many
SELECT
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at
FROM agents
WHERE status = sqlc.arg(status) AND cleanup_state = 'active'
ORDER BY updated_at DESC, id ASC;

-- name: GetAgentByID :one
SELECT
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at
FROM agents
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: GetAgentByTmuxSession :one
SELECT
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at
FROM agents
WHERE tmux_session = sqlc.arg(tmux_session)
LIMIT 1;

-- name: CreateAgent :one
INSERT INTO agents (
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state
) VALUES (
    sqlc.arg(name),
    sqlc.arg(tmux_session),
    sqlc.arg(status),
    sqlc.narg(task),
    sqlc.narg(last_activity),
    sqlc.narg(branch),
    sqlc.narg(working_dir),
    sqlc.narg(profile_name),
    sqlc.narg(ticket_id),
    sqlc.narg(initial_prompt),
    sqlc.narg(last_heartbeat_at),
    sqlc.narg(last_error),
    COALESCE(sqlc.narg(cleanup_state), 'active')
)
RETURNING
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at;

-- name: UpdateAgent :one
UPDATE agents
SET
    name = sqlc.arg(name),
    tmux_session = sqlc.arg(tmux_session),
    status = sqlc.arg(status),
    task = sqlc.narg(task),
    last_activity = sqlc.narg(last_activity),
    branch = sqlc.narg(branch),
    working_dir = sqlc.narg(working_dir),
    profile_name = sqlc.narg(profile_name),
    ticket_id = sqlc.narg(ticket_id),
    initial_prompt = sqlc.narg(initial_prompt),
    last_heartbeat_at = sqlc.narg(last_heartbeat_at),
    last_error = sqlc.narg(last_error),
    cleanup_state = COALESCE(sqlc.narg(cleanup_state), cleanup_state),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING
    id,
    name,
    tmux_session,
    status,
    task,
    last_activity,
    branch,
    working_dir,
    profile_name,
    ticket_id,
    initial_prompt,
    last_heartbeat_at,
    last_error,
    cleanup_state,
    created_at,
    updated_at;

-- name: UpdateAgentStatusByID :exec
UPDATE agents
SET
    status = sqlc.arg(status),
    task = COALESCE(sqlc.narg(task), task),
    last_activity = COALESCE(sqlc.narg(last_activity), last_activity),
    branch = COALESCE(sqlc.narg(branch), branch),
    last_error = sqlc.narg(last_error),
    last_heartbeat_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: UpdateAgentStatusByTmuxSession :exec
UPDATE agents
SET
    status = sqlc.arg(status),
    task = COALESCE(sqlc.narg(task), task),
    last_activity = COALESCE(sqlc.narg(last_activity), last_activity),
    branch = COALESCE(sqlc.narg(branch), branch),
    last_error = sqlc.narg(last_error),
    last_heartbeat_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE tmux_session = sqlc.arg(tmux_session);

-- name: UpdateAgentCleanupState :exec
UPDATE agents
SET
    cleanup_state = sqlc.arg(cleanup_state),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: DeleteAgent :exec
DELETE FROM agents
WHERE id = sqlc.arg(id);
