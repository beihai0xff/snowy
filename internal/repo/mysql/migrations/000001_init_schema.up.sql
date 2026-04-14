-- Snowy 初始化 Schema (MySQL 8.0+)
-- 参考技术方案 §18.1 & §18.2

-- ── 用户 ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    phone           VARCHAR(20)  NOT NULL UNIQUE,
    nickname        VARCHAR(64)  NOT NULL DEFAULT '',
    role            VARCHAR(16)  NOT NULL DEFAULT 'student',
    avatar_url      TEXT         NOT NULL,
    last_login_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_users_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 收藏 ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS favorites (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    target_type     VARCHAR(32)  NOT NULL,
    target_id       VARCHAR(64)  NOT NULL,
    title           TEXT         NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_favorites_user (user_id, created_at DESC),
    CONSTRAINT fk_favorites_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── Agent 会话 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS agent_sessions (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    mode            VARCHAR(32)  NOT NULL DEFAULT 'auto',
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    metadata        JSON,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_agent_sessions_user (user_id, created_at DESC),
    CONSTRAINT fk_agent_sessions_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── Agent 消息 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS agent_messages (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    session_id      CHAR(36)     NOT NULL,
    role            VARCHAR(16)  NOT NULL,
    content         TEXT         NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_agent_messages_session (session_id, created_at ASC),
    CONSTRAINT fk_agent_messages_session FOREIGN KEY (session_id) REFERENCES agent_sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── Agent 运行记录 ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS agent_runs (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    session_id      CHAR(36)     NOT NULL,
    message_id      CHAR(36)     NOT NULL,
    mode            VARCHAR(32)  NOT NULL,
    model_name      VARCHAR(64)  NOT NULL,
    prompt_version  VARCHAR(32)  NOT NULL DEFAULT '',
    input_tokens    INT          NOT NULL DEFAULT 0,
    output_tokens   INT          NOT NULL DEFAULT 0,
    estimated_cost  DECIMAL(10,6) NOT NULL DEFAULT 0,
    latency_ms      INT          NOT NULL DEFAULT 0,
    confidence      DECIMAL(4,3) NOT NULL DEFAULT 0,
    fallback_reason VARCHAR(128) DEFAULT '',
    status          VARCHAR(16)  NOT NULL DEFAULT 'success',
    error_code      VARCHAR(64)  DEFAULT '',
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_agent_runs_session (session_id, created_at DESC),
    CONSTRAINT fk_agent_runs_session FOREIGN KEY (session_id) REFERENCES agent_sessions(id),
    CONSTRAINT fk_agent_runs_message FOREIGN KEY (message_id) REFERENCES agent_messages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── Agent 工具调用记录 ──────────────────────────────────
CREATE TABLE IF NOT EXISTS agent_tool_calls (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    run_id          CHAR(36)     NOT NULL,
    tool_name       VARCHAR(64)  NOT NULL,
    input           JSON,
    output          JSON,
    latency_ms      INT          NOT NULL DEFAULT 0,
    status          VARCHAR(16)  NOT NULL DEFAULT 'success',
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_agent_tool_calls_run (run_id),
    CONSTRAINT fk_agent_tool_calls_run FOREIGN KEY (run_id) REFERENCES agent_runs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 内容文档 ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS content_documents (
    id               CHAR(36)     NOT NULL PRIMARY KEY,
    doc_id           VARCHAR(128) NOT NULL UNIQUE,
    source_type      VARCHAR(32)  NOT NULL,
    subject          VARCHAR(32)  NOT NULL,
    grade            VARCHAR(32)  NOT NULL DEFAULT '',
    chapter          VARCHAR(128) NOT NULL DEFAULT '',
    section          VARCHAR(128) NOT NULL DEFAULT '',
    knowledge_tags   JSON         NOT NULL,
    topic_tags       JSON         NOT NULL,
    difficulty       VARCHAR(16)  DEFAULT '',
    content          LONGTEXT     NOT NULL,
    answer           LONGTEXT,
    metadata         JSON,
    copyright_status VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    version          INT          NOT NULL DEFAULT 1,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_content_docs_subject (subject),
    INDEX idx_content_docs_source (source_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 内容切片 ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS content_chunks (
    id               CHAR(36)     NOT NULL PRIMARY KEY,
    document_id      CHAR(36)     NOT NULL,
    chunk_index      INT          NOT NULL,
    content          LONGTEXT     NOT NULL,
    tags             JSON         NOT NULL,
    chunk_type       VARCHAR(32)  NOT NULL DEFAULT 'paragraph',
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_content_chunks_doc (document_id, chunk_index),
    CONSTRAINT fk_content_chunks_doc FOREIGN KEY (document_id) REFERENCES content_documents(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 检索日志 ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS search_logs (
    id               CHAR(36)     NOT NULL PRIMARY KEY DEFAULT (UUID()),
    query_text       TEXT         NOT NULL,
    user_id          VARCHAR(64)  DEFAULT '',
    result_count     INT          NOT NULL DEFAULT 0,
    latency_ms       INT          NOT NULL DEFAULT 0,
    top_score        DECIMAL(5,4) NOT NULL DEFAULT 0,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 物理建模运行 ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS physics_runs (
    id               CHAR(36)     NOT NULL PRIMARY KEY,
    session_id       CHAR(36)     NOT NULL,
    question         TEXT         NOT NULL,
    result           JSON,
    model_name       VARCHAR(64)  NOT NULL DEFAULT '',
    latency_ms       INT          NOT NULL DEFAULT 0,
    status           VARCHAR(16)  NOT NULL DEFAULT 'success',
    fallback_reason  VARCHAR(128) DEFAULT '',
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 生物建模运行 ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS biology_runs (
    id               CHAR(36)     NOT NULL PRIMARY KEY,
    session_id       CHAR(36)     NOT NULL,
    question         TEXT         NOT NULL,
    result           JSON,
    model_name       VARCHAR(64)  NOT NULL DEFAULT '',
    latency_ms       INT          NOT NULL DEFAULT 0,
    status           VARCHAR(16)  NOT NULL DEFAULT 'success',
    fallback_reason  VARCHAR(128) DEFAULT '',
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 概念图快照 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS concept_graph_snapshots (
    id               CHAR(36)     NOT NULL PRIMARY KEY DEFAULT (UUID()),
    biology_run_id   CHAR(36),
    diagram          JSON         NOT NULL,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_concept_graph_biology FOREIGN KEY (biology_run_id) REFERENCES biology_runs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── Prompt 版本管理 ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS prompt_templates (
    id               CHAR(36)     NOT NULL PRIMARY KEY DEFAULT (UUID()),
    scene            VARCHAR(64)  NOT NULL,
    version          VARCHAR(32)  NOT NULL,
    template         LONGTEXT     NOT NULL,
    status           VARCHAR(16)  NOT NULL DEFAULT 'active',
    traffic_pct      INT          NOT NULL DEFAULT 100,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_scene_version (scene, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

