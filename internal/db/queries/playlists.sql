-- name: GetPlaylist :one
SELECT * FROM playlists
WHERE id = $1;

-- name: ListPlaylistsByOwner :many
SELECT
    playlists.*,
    COUNT(playlist_tracks.track_id)::int AS song_count,
    COALESCE(SUM(tracks.duration_seconds), 0)::int AS duration_seconds
FROM playlists
LEFT JOIN playlist_tracks ON playlist_tracks.playlist_id = playlists.id
LEFT JOIN tracks ON tracks.id = playlist_tracks.track_id
WHERE playlists.owner_id = $1
GROUP BY playlists.id
ORDER BY playlists.name;

-- name: CreatePlaylist :one
INSERT INTO playlists (name, owner_id, public)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdatePlaylist :one
UPDATE playlists
SET name = $2, public = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePlaylist :exec
DELETE FROM playlists
WHERE id = $1;
