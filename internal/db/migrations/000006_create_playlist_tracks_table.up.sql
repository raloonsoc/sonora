CREATE TABLE playlist_tracks (
    playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id    UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    position    INT NOT NULL,
    PRIMARY KEY (playlist_id, position)
);

CREATE INDEX idx_playlist_tracks_track_id ON playlist_tracks(track_id);
