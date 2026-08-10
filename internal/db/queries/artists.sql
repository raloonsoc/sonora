-- name: GetArtist :one
SELECT * FROM artists
WHERE id = $1;

-- name: GetArtistByName :one
SELECT * FROM artists
WHERE name = $1;

-- name: ListArtists :many
SELECT DISTINCT artists.* FROM artists
JOIN track_artists ON track_artists.artist_id = artists.id
ORDER BY name;

-- name: SearchArtists :many
SELECT DISTINCT artists.* FROM artists
JOIN track_artists ON track_artists.artist_id = artists.id
WHERE artists.name ILIKE '%' || $1 || '%'
ORDER BY name
LIMIT $2;

-- name: CreateArtist :one
INSERT INTO artists (name)
VALUES ($1)
RETURNING *;

-- name: DeleteArtist :exec
DELETE FROM artists
WHERE id = $1;
