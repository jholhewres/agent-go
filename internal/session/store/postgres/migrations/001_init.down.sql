-- 001_init.down.sql
-- Reverses the initial schema migration.

DROP INDEX IF EXISTS idx_agno_sessions_workflow;
DROP INDEX IF EXISTS idx_agno_sessions_type_agent;
DROP INDEX IF EXISTS idx_agno_sessions_user_updated;
DROP TABLE IF EXISTS agno_sessions;
