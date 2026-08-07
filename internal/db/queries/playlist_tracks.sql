-- name: ListPlaylistTracks :many
SELECT * FROM playlist_tracks
WHERE playlist_id = $1
ORDER BY position;

-- name: AddPlaylistTrack :one
INSERT INTO playlist_tracks (playlist_id, track_id, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemovePlaylistTrack :exec
DELETE FROM playlist_tracks
WHERE playlist_id = $1 AND track_id = $2;

-- name: ClearPlaylistTracks :exec
DELETE FROM playlist_tracks
WHERE playlist_id = $1;
