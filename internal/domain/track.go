package domain

type Track struct {
	ID                string
	Title             string
	AlbumID           string
	ArtistID          string
	Genre             string
	TrackNumber       int
	DiscNumber        int
	DurationSeconds   int // seconds not ms
	Path              string
	Format            string
	ReplayGainTrackDB *float64
}
