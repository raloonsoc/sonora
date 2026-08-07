CREATE TABLE tracks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                TEXT NOT NULL,
    album_id             UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    artist_id            UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    genre                TEXT NOT NULL DEFAULT '',
    track_number         INT NOT NULL DEFAULT 0,
    disc_number          INT NOT NULL DEFAULT 0,
    duration_seconds     INT NOT NULL,
    path                 TEXT NOT NULL UNIQUE,
    format               TEXT NOT NULL,
    replay_gain_track_db DOUBLE PRECISION,
    bit_depth            INT NOT NULL DEFAULT 0,
    sample_rate          INT NOT NULL DEFAULT 0,
    channels             INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_tracks_album_id ON tracks(album_id);
CREATE INDEX idx_tracks_artist_id ON tracks(artist_id);
