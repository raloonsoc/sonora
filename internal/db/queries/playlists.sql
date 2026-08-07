-- name: GetPlaylist :one
SELECT * FROM playlists
WHERE id = $1;

-- name: ListPlaylistsByOwner :many
SELECT * FROM playlists
WHERE owner_id = $1
ORDER BY name;

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
