-- 001_init.up.sql
-- Initial schema for the agno session store.

CREATE TABLE IF NOT EXISTS agno_sessions (
    session_id   TEXT        PRIMARY KEY,
    session_type TEXT        NOT NULL,
    agent_id     TEXT,
    team_id      TEXT,
    workflow_id  TEXT,
    user_id      TEXT,
    session_data JSONB,
    agent_data   JSONB,
    team_data    JSONB,
    workflow_data JSONB,
    metadata     JSONB,
    runs         JSONB,
    summary      JSONB,
    created_at   BIGINT      NOT NULL,
    updated_at   BIGINT
);

-- Index for listing sessions by user ordered by recency (primary tenant query).
CREATE INDEX IF NOT EXISTS idx_agno_sessions_user_updated
    ON agno_sessions (user_id, updated_at DESC);

-- Index for filtering by session type + component (agent/team/workflow).
CREATE INDEX IF NOT EXISTS idx_agno_sessions_type_agent
    ON agno_sessions (session_type, agent_id);

-- Index for non-null workflow_id lookups.
CREATE INDEX IF NOT EXISTS idx_agno_sessions_workflow
    ON agno_sessions (workflow_id)
    WHERE workflow_id IS NOT NULL;
