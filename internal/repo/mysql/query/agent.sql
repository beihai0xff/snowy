-- name: CreateAgentSession :exec
INSERT INTO agent_sessions (id, user_id, mode, status, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentSession :one
SELECT id, user_id, mode, status, metadata, created_at, updated_at
FROM agent_sessions WHERE id = ?;

-- name: ListAgentSessionsByUser :many
SELECT id, user_id, mode, status, metadata, created_at, updated_at
FROM agent_sessions WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountAgentSessionsByUser :one
SELECT COUNT(*) FROM agent_sessions WHERE user_id = ?;

-- name: UpdateAgentSessionStatus :exec
UPDATE agent_sessions SET status = ?, updated_at = NOW() WHERE id = ?;

-- name: CreateAgentMessage :exec
INSERT INTO agent_messages (id, session_id, role, content, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListAgentMessages :many
SELECT id, session_id, role, content, created_at
FROM agent_messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?;

-- name: CountAgentMessages :one
SELECT COUNT(*) FROM agent_messages WHERE session_id = ?;

-- name: CreateAgentRun :exec
INSERT INTO agent_runs (id, session_id, message_id, mode, model_name, prompt_version,
    input_tokens, output_tokens, estimated_cost, latency_ms, confidence,
    fallback_reason, status, error_code, created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);

-- name: GetAgentRun :one
SELECT id, session_id, message_id, mode, model_name, prompt_version,
    input_tokens, output_tokens, estimated_cost, latency_ms, confidence,
    fallback_reason, status, error_code, created_at
FROM agent_runs WHERE id = ?;

-- name: CreateAgentToolCall :exec
INSERT INTO agent_tool_calls (id, run_id, tool_name, input, output, latency_ms, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAgentToolCallsByRun :many
SELECT id, run_id, tool_name, input, output, latency_ms, status, created_at
FROM agent_tool_calls WHERE run_id = ?;

