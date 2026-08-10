ALTER TABLE tracks
    ADD COLUMN fingerprint TEXT NOT NULL DEFAULT '';

-- Chromaprint fingerprints are long (several KB as base64), too large for
-- a plain btree index (Postgres row size limit). md5() gives a fixed-size
-- 32-char digest that's cheap to index and exact-match lookups on it are
-- equivalent to matching the full fingerprint.
CREATE INDEX idx_tracks_fingerprint_md5 ON tracks(md5(fingerprint)) WHERE fingerprint != '';
