CREATE TABLE track_artists (
    track_id  UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    position  INT NOT NULL DEFAULT 0,
    PRIMARY KEY (track_id, artist_id)
);

CREATE INDEX idx_track_artists_artist_id ON track_artists(artist_id);

-- Backfill: every existing track keeps its current single artist as
-- position 0. Composite names already stored as one artist (e.g. "aespa &
-- Ty Dolla $ign") are not split retroactively — only future ingests use
-- the new parser. Re-ingesting a file will pick up the split automatically.
INSERT INTO track_artists (track_id, artist_id, position)
SELECT id, artist_id, 0 FROM tracks;
