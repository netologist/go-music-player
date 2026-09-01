package musicplayer

import "time"

// Song is the fundamental domain unit of the system.
// The ID field is used as the unique identifier (reference point for dedup, lookup, and merge policies).
// Duration is stored for future use in shuffle/statistics features.
type Song struct {
	ID       string // unique identifier (e.g. UUID or catalog ID)
	Title    string
	Artist   string
	Duration time.Duration
}

// NewSong is a simple constructor. It prevents creating a song with an empty ID
// because the ID is the single source of truth that dedup and merge logic is built upon.
func NewSong(id, title, artist string, duration time.Duration) (Song, error) {
	if id == "" {
		return Song{}, ErrEmptySongID
	}
	return Song{
		ID:       id,
		Title:    title,
		Artist:   artist,
		Duration: duration,
	}, nil
}
