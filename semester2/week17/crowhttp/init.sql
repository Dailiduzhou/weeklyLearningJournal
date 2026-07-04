-- Init schema for the crowhttp demo.
-- Runs automatically by the postgres image on first startup against POSTGRES_DB.

CREATE TABLE IF NOT EXISTS users (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    age  INTEGER      NOT NULL
);

INSERT INTO users (name, age) VALUES
    ('Alice',   30),
    ('Bob',     25),
    ('Charlie', 35)
ON CONFLICT DO NOTHING;
