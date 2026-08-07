-- name: GetAlbum :one
SELECT * FROM albums
WHERE id = $1;

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
