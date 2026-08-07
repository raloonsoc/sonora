package db

import (
	"github.com/raloonsoc/sonora/internal/db/sqlc"
	"github.com/raloonsoc/sonora/internal/domain"
)

func trackFromSQLC(t sqlc.Track) domain.Track {
	var replayGain *float64
	if t.ReplayGainTrackDb.Valid {
		replayGain = &t.ReplayGainTrackDb.Float64
	}

	track := domain.Track{
		ID:                t.ID.String(),
		Title:             t.Title,
		AlbumID:           t.AlbumID.String(),
		ArtistID:          t.ArtistID.String(),
		Genre:             t.Genre,
		TrackNumber:       int(t.TrackNumber),
		DiscNumber:        int(t.DiscNumber),
		DurationSeconds:   int(t.DurationSeconds),
		Path:              t.Path,
		Format:            t.Format,
		ReplayGainTrackDB: replayGain,
	}

	return track
}

func albumFromSQLC(t sqlc.Album) domain.Album {
	var releaseYear *int
	if t.ReleaseYear.Valid {
		year := int(t.ReleaseYear.Int32)
		releaseYear = &year
	}
	album := domain.Album{
		ID:           t.ID.String(),
		Title:        t.Title,
		ArtistID:     t.ArtistID.String(),
		ReleaseYear:  releaseYear,
		CoverArtPath: t.CoverArtPath,
		CreatedAt:    t.CreatedAt.Time,
	}

	return album
}

func artistFromSQLC(t sqlc.Artist) domain.Artist {
	artist := domain.Artist{
		ID:   t.ID.String(),
		Name: t.Name,
	}

	return artist
}

func playlistFromSQLC(t sqlc.Playlist) domain.Playlist {
	playlist := domain.Playlist{
		ID:        t.ID.String(),
		Name:      t.Name,
		OwnerID:   t.OwnerID.String(),
		Public:    t.Public,
		CreatedAt: t.CreatedAt.Time,
		UpdatedAt: t.UpdatedAt.Time,
	}

	return playlist
}

func userFromSQLC(t sqlc.User) domain.User {
	user := domain.User{
		ID:                t.ID.String(),
		Username:          t.Username,
		PasswordEncrypted: t.PasswordEncrypted,
		IsAdmin:           t.IsAdmin,
		CreatedAt:         t.CreatedAt.Time,
	}

	return user
}
