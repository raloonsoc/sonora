-- name: GetTrack :one
SELECT * FROM tracks
WHERE id = $1;

-- name: GetTrackByPath :one
SELECT * FROM tracks
WHERE path = $1;

-- name: ListTrackPaths :many
SELECT path FROM tracks;

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

-- name: ListGenres :many
SELECT genre, COUNT(*) AS song_count, COUNT(DISTINCT album_id) AS album_count
FROM tracks
WHERE genre != ''
GROUP BY genre
ORDER BY genre;

-- name: CreateTrack :one
INSERT INTO tracks (
    title, album_id, artist_id, genre, track_number, disc_number,
    duration_seconds, path, format, replay_gain_track_db,
    bit_depth, sample_rate, channels, bit_rate, size_bytes
)
VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13, $14, $15
)
RETURNING *;

-- name: DeleteTrack :exec
DELETE FROM tracks
WHERE id = $1;

-- name: ScrobbleTrack :exec
UPDATE tracks
SET play_count = play_count + 1,
    last_played_at = COALESCE(sqlc.narg(played_at), NOW())
WHERE id = sqlc.arg(id);
