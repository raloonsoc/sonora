CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE artists (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);
