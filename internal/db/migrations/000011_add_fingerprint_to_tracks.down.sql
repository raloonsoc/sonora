DROP INDEX idx_tracks_fingerprint_md5;

ALTER TABLE tracks
    DROP COLUMN fingerprint;
