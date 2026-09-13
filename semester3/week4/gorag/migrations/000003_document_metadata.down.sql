DROP INDEX IF EXISTS documents_tags_idx;
DROP INDEX IF EXISTS documents_metadata_idx;
ALTER TABLE documents DROP COLUMN IF EXISTS tags;
ALTER TABLE documents DROP COLUMN IF EXISTS metadata;