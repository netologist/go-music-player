package musicplayer

import (
	"errors"
	"time"
)

// SongBuilder builds Song domain objects step-by-step with a fluent interface
// (Builder Pattern).
//
// Why Builder Pattern:
// It simplifies construction of objects with many parameters, supplies sensible
// defaults, and makes test fixtures far more readable and flexible.
type SongBuilder struct {
	id       string
	title    string
	artist   string
	duration time.Duration
	err      error
}

// NewSongBuilder initialises a new SongBuilder with the given ID.
func NewSongBuilder(id string) *SongBuilder {
	b := &SongBuilder{
		id:       id,
		duration: 3 * time.Minute, // sensible default duration
		artist:   "Unknown Artist",
		title:    "Untitled",
	}
	if id == "" {
		b.err = ErrEmptySongID
	}
	return b
}

// Title sets the song title.
func (b *SongBuilder) Title(title string) *SongBuilder {
	b.title = title
	return b
}

// Artist sets the song artist.
func (b *SongBuilder) Artist(artist string) *SongBuilder {
	b.artist = artist
	return b
}

// Duration sets the song duration.
func (b *SongBuilder) Duration(d time.Duration) *SongBuilder {
	b.duration = d
	return b
}

// Build validates and produces the configured Song.
func (b *SongBuilder) Build() (Song, error) {
	if b.err != nil {
		return Song{}, b.err
	}
	return NewSong(b.id, b.title, b.artist, b.duration)
}

// MustBuild panics on error. Intended for test fixtures and demos.
func (b *SongBuilder) MustBuild() Song {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}
	return s
}

// PlaylistBuilder builds Playlist objects and their initial songs with a fluent
// interface (Builder Pattern).
type PlaylistBuilder struct {
	name         string
	dedupEnabled bool
	songs        []Song
	errors       []error
}

// NewPlaylistBuilder initialises a PlaylistBuilder with the given name.
func NewPlaylistBuilder(name string) *PlaylistBuilder {
	return &PlaylistBuilder{
		name: name,
	}
}

// WithDedup configures duplicate-song detection for the playlist.
func (b *PlaylistBuilder) WithDedup(enabled bool) *PlaylistBuilder {
	b.dedupEnabled = enabled
	return b
}

// AddSong appends an existing Song to the builder's list.
func (b *PlaylistBuilder) AddSong(s Song) *PlaylistBuilder {
	b.songs = append(b.songs, s)
	return b
}

// AddNewSong creates a Song from raw fields and appends it to the builder's list.
func (b *PlaylistBuilder) AddNewSong(id, title, artist string, duration time.Duration) *PlaylistBuilder {
	s, err := NewSong(id, title, artist, duration)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.songs = append(b.songs, s)
	return b
}

// Build produces the configured Playlist and checks for accumulated errors.
func (b *PlaylistBuilder) Build() (*Playlist, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}
	p := NewPlaylist(b.name, WithDedup(b.dedupEnabled))
	for _, s := range b.songs {
		if err := p.AddSong(s); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// MustBuild panics on error.
func (b *PlaylistBuilder) MustBuild() *Playlist {
	p, err := b.Build()
	if err != nil {
		panic(err)
	}
	return p
}
