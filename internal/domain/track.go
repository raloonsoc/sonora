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
	BitRate           int
	SizeBytes         int64
	Path              string
	Format            string
	ReplayGainTrackDB *float64
}
