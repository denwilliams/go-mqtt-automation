-- Down migration - drop all tables and data

DROP INDEX IF EXISTS idx_state_updated;
DROP INDEX IF EXISTS idx_execution_log_executed;
DROP INDEX IF EXISTS idx_execution_log_topic;
DROP INDEX IF EXISTS idx_strategies_name;
DROP INDEX IF EXISTS idx_topics_updated;
DROP INDEX IF EXISTS idx_topics_type;

DROP TABLE IF EXISTS execution_log;
DROP TABLE IF EXISTS state;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS strategies;