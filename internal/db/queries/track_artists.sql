-- name: CreateTrackArtist :exec
INSERT INTO track_artists (track_id, artist_id, position)
VALUES ($1, $2, $3);

-- name: ListArtistsByTrack :many
SELECT artists.*
FROM artists
JOIN track_artists ON track_artists.artist_id = artists.id
WHERE track_artists.track_id = $1
ORDER BY track_artists.position;

-- name: ListTracksByArtistIDIncludingFeatured :many
SELECT DISTINCT tracks.*
FROM tracks
JOIN track_artists ON track_artists.track_id = tracks.id
WHERE track_artists.artist_id = $1
ORDER BY tracks.title;
