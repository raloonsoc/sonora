CREATE TABLE albums (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title          TEXT NOT NULL,
    artist_id      UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    release_year   INT,
    cover_art_path TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_albums_artist_id ON albums(artist_id);
