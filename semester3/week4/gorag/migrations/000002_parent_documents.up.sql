-- Parent Document Retrieval stores each document's whole cleaned text beside
-- the version being activated. Child chunks stay the searchable unit while
-- retrieval can expand a hit back to its full parent document.
CREATE TABLE document_parents (
    document_id BIGINT PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    document_version TEXT NOT NULL CHECK (document_version <> ''),
    content TEXT NOT NULL CHECK (content <> ''),
    start_line INTEGER NOT NULL CHECK (start_line > 0),
    end_line INTEGER NOT NULL CHECK (end_line >= start_line),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);