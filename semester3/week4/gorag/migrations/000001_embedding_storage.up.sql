CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    source_path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    current_version TEXT,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at TIMESTAMPTZ,
    CHECK (current_version IS NULL OR current_version <> '')
);

CREATE TABLE document_chunks (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    document_version TEXT NOT NULL CHECK (document_version <> ''),
    chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
    content TEXT NOT NULL,
    heading_path TEXT[] NOT NULL DEFAULT '{}',
    start_line INTEGER NOT NULL CHECK (start_line > 0),
    end_line INTEGER NOT NULL CHECK (end_line >= start_line),
    content_hash TEXT NOT NULL,
    embedding_model TEXT NOT NULL
        CHECK (embedding_model = 'qwen3-embedding:0.6b'),
    embedding_dimension INTEGER NOT NULL
        CHECK (embedding_dimension = 1024),
    embedding vector(1024) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_id, document_version, chunk_index)
);

CREATE TABLE index_runs (
    id BIGSERIAL PRIMARY KEY,
    run_type TEXT NOT NULL CHECK (run_type IN ('sync', 'add', 'delete', 'reindex')),
    status TEXT NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    document_count INTEGER NOT NULL DEFAULT 0 CHECK (document_count >= 0),
    chunk_count INTEGER NOT NULL DEFAULT 0 CHECK (chunk_count >= 0),
    error_message TEXT,
    CHECK (
        (status = 'running' AND completed_at IS NULL)
        OR (status IN ('completed', 'failed') AND completed_at IS NOT NULL)
    ),
    CHECK (
        (status = 'failed' AND error_message IS NOT NULL)
        OR (status <> 'failed' AND error_message IS NULL)
    )
);

CREATE INDEX document_chunks_document_version_idx
    ON document_chunks (document_id, document_version);

CREATE INDEX documents_status_current_version_idx
    ON documents (status, current_version);
