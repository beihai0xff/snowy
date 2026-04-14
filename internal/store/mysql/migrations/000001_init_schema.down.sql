-- Rollback: 按依赖反序删除
DROP TABLE IF EXISTS prompt_templates;
DROP TABLE IF EXISTS concept_graph_snapshots;
DROP TABLE IF EXISTS biology_runs;
DROP TABLE IF EXISTS physics_runs;
DROP TABLE IF EXISTS search_logs;
DROP TABLE IF EXISTS content_chunks;
DROP TABLE IF EXISTS content_documents;
DROP TABLE IF EXISTS agent_tool_calls;
DROP TABLE IF EXISTS agent_runs;
DROP TABLE IF EXISTS agent_messages;
DROP TABLE IF EXISTS agent_sessions;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS users;

