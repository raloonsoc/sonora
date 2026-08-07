package domain

import "time"

type Playlist struct {
	ID        string
	Name      string
	OwnerID   string
	Public    bool
	TrackIDs  []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
