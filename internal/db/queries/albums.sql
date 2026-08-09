-- name: GetAlbum :one
SELECT * FROM albums
WHERE id = $1;

-- name: GetAlbumWithStats :one
SELECT
    albums.*,
    COALESCE(SUM(tracks.duration_seconds), 0)::int AS duration_seconds,
    COALESCE(SUM(tracks.play_count), 0)::int AS play_count,
    COUNT(tracks.id)::int AS song_count,
    COALESCE(MODE() WITHIN GROUP (ORDER BY tracks.genre), '')::text AS genre
FROM albums
LEFT JOIN tracks ON tracks.album_id = albums.id
WHERE albums.id = $1
GROUP BY albums.id;

-- name: GetAlbumByTitleAndArtist :one
SELECT * FROM albums
WHERE title = $1 AND artist_id = $2;

-- name: ListAlbumsByArtist :many
SELECT * FROM albums
WHERE artist_id = $1
ORDER BY release_year, title;

-- name: SearchAlbums :many
SELECT * FROM albums
WHERE title ILIKE '%' || $1 || '%'
ORDER BY title
LIMIT $2;

-- name: ListAlbumsAlphabetical :many
SELECT
    albums.*,
    COALESCE(SUM(tracks.duration_seconds), 0)::int AS duration_seconds,
    COALESCE(SUM(tracks.play_count), 0)::int AS play_count,
    COUNT(tracks.id)::int AS song_count,
    COALESCE(MODE() WITHIN GROUP (ORDER BY tracks.genre), '')::text AS genre
FROM albums
LEFT JOIN tracks ON tracks.album_id = albums.id
GROUP BY albums.id
ORDER BY albums.title
LIMIT $1 OFFSET $2;

-- name: ListAlbumsNewest :many
SELECT
    albums.*,
    COALESCE(SUM(tracks.duration_seconds), 0)::int AS duration_seconds,
    COALESCE(SUM(tracks.play_count), 0)::int AS play_count,
    COUNT(tracks.id)::int AS song_count,
    COALESCE(MODE() WITHIN GROUP (ORDER BY tracks.genre), '')::text AS genre
FROM albums
LEFT JOIN tracks ON tracks.album_id = albums.id
GROUP BY albums.id
ORDER BY albums.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateAlbumCoverArt :exec
UPDATE albums
SET cover_art_path = $2
WHERE id = $1;

-- name: CreateAlbum :one
INSERT INTO albums (title, artist_id, release_year, cover_art_path)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteAlbum :exec
DELETE FROM albums
WHERE id = $1;
