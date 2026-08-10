-- name: StarItem :exec
INSERT INTO starred_items (user_id, item_type, item_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, item_type, item_id) DO NOTHING;

-- name: UnstarItem :exec
DELETE FROM starred_items
WHERE user_id = $1 AND item_type = $2 AND item_id = $3;

-- name: ListStarredTracks :many
SELECT tracks.*, starred_items.starred_at
FROM tracks
JOIN starred_items ON starred_items.item_type = 'track' AND starred_items.item_id = tracks.id
WHERE starred_items.user_id = $1
ORDER BY starred_items.starred_at;

-- name: ListStarredAlbums :many
SELECT
    albums.*,
    starred_items.starred_at,
    COALESCE(SUM(tracks.duration_seconds), 0)::int AS duration_seconds,
    COALESCE(SUM(tracks.play_count), 0)::int AS play_count,
    COUNT(tracks.id)::int AS song_count,
    COALESCE(MODE() WITHIN GROUP (ORDER BY tracks.genre), '')::text AS genre
FROM albums
JOIN starred_items ON starred_items.item_type = 'album' AND starred_items.item_id = albums.id
LEFT JOIN tracks ON tracks.album_id = albums.id
WHERE starred_items.user_id = $1
GROUP BY albums.id, starred_items.starred_at
ORDER BY starred_items.starred_at;

-- name: ListStarredArtists :many
SELECT artists.*, starred_items.starred_at
FROM artists
JOIN starred_items ON starred_items.item_type = 'artist' AND starred_items.item_id = artists.id
WHERE starred_items.user_id = $1
ORDER BY starred_items.starred_at;
