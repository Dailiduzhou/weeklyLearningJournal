-- Filterable document metadata parsed from front matter. Scalar keys live in
-- a JSONB object; the tags list gets a first-class array column so filters
-- can use array overlap. Rows are written by ActivateVersion in the same
-- transaction as the version flip.
ALTER TABLE documents ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE documents ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';

-- Both columns are filtered with containment/overlap operators, which only
-- benefit from GIN indexes.
CREATE INDEX documents_metadata_idx ON documents USING GIN (metadata jsonb_path_ops);
CREATE INDEX documents_tags_idx ON documents USING GIN (tags);