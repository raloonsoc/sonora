package domain

import "time"

type Album struct {
	ID           string
	Title        string
	ArtistID     string
	ReleaseYear  *int
	CoverArtPath string
	CreatedAt    time.Time
}
