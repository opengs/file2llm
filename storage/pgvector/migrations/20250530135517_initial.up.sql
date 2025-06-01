CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE SCHEMA_NAME.DATABASE_PREFIX_processor_version AS (
    major INTEGER,
    minor INTEGER,
    patch INTEGER,
    model TEXT
);

--CREATE TYPE SCHEMA_NAME.DATABASE_PREFIX_filepart_parse_error AS (
--    part TEXT,
--    error TEXT
--);

CREATE TABLE IF NOT EXISTS SCHEMA_NAME.DATABASE_PREFIX_source (
    source_id SERIAL PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS SCHEMA_NAME.DATABASE_PREFIX_file (
    file_id BIGSERIAL,
    source_id INTEGER REFERENCES SCHEMA_NAME.DATABASE_PREFIX_source(source_id) ON DELETE CASCADE,
    etag TEXT NOT NULL,
    path TEXT NOT NULL,
    parsed BOOLEAN NOT NULL DEFAULT FALSE,
    parse_error TEXT,
    parse_parts_errors TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processor_version SCHEMA_NAME.DATABASE_PREFIX_processor_version NOT NULL,
    processing_finished TIMESTAMPTZ,
    PRIMARY KEY (source_id, file_id),
    UNIQUE (source_id, path)
) PARTITION BY LIST (source_id);

CREATE TABLE SCHEMA_NAME.DATABASE_PREFIX_file_default PARTITION OF SCHEMA_NAME.DATABASE_PREFIX_file DEFAULT;

CREATE TABLE IF NOT EXISTS SCHEMA_NAME.DATABASE_PREFIX_embedding (
    embedding_id BIGSERIAL,
    source_id INTEGER REFERENCES SCHEMA_NAME.DATABASE_PREFIX_source(source_id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL,
    chunk TEXT NOT NULL,
    embedding vector(VECTOR_DIMENSIONS),
    PRIMARY KEY (source_id, embedding_id),
    FOREIGN KEY (source_id, file_id) REFERENCES SCHEMA_NAME.DATABASE_PREFIX_file(source_id, file_id) ON DELETE CASCADE
) PARTITION BY LIST (source_id);

CREATE TABLE SCHEMA_NAME.DATABASE_PREFIX_embedding_default PARTITION OF SCHEMA_NAME.DATABASE_PREFIX_embedding DEFAULT;
CREATE INDEX IF NOT EXISTS idx_DATABASE_PREFIX_embedding_default_vector ON SCHEMA_NAME.DATABASE_PREFIX_embedding_default USING hnsw (embedding vector_ip_ops); -- vectors are normalized