CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username           TEXT NOT NULL UNIQUE,
    password_encrypted BYTEA NOT NULL,
    is_admin           BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
