-- Remove tags column from topics table
DROP INDEX IF EXISTS idx_topics_tags;
ALTER TABLE topics DROP COLUMN tags;
