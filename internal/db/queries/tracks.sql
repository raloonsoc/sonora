-- name: GetTrack :one
SELECT * FROM tracks
WHERE id = $1;

-- name: GetTrackByPath :one
SELECT * FROM tracks
WHERE path = $1;

-- name: ListTracksByAlbum :many
SELECT * FROM tracks
WHERE album_id = $1
ORDER BY disc_number, track_number;

-- name: ListTracksByArtist :many
SELECT * FROM tracks
WHERE artist_id = $1
ORDER BY title;

-- name: SearchTracks :many
SELECT * FROM tracks
WHERE title ILIKE '%' || $1 || '%'
ORDER BY title
LIMIT $2;

-- name: CreateTrack :one
INSERT INTO tracks (
    title, album_id, artist_id, genre, track_number, disc_number,
    duration_seconds, path, format, replay_gain_track_db,
    bit_depth, sample_rate, channels
)
VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13
)
RETURNING *;

-- name: DeleteTrack :exec
DELETE FROM tracks
WHERE id = $1;
