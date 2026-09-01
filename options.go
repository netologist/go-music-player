package musicplayer

import "math/rand/v2"

// PlaylistOption is the functional option type used to configure a Playlist.
// Functional Options Pattern: avoids constructor parameter explosion and the
// "boolean trap" anti-pattern that hurts readability.
type PlaylistOption func(*Playlist)

// WithDedup enables or disables duplicate song detection for the playlist.
func WithDedup(enabled bool) PlaylistOption {
	return func(p *Playlist) {
		p.DedupEnabled = enabled
	}
}

// WithInitialSongs adds a set of initial songs when the playlist is created.
func WithInitialSongs(songs ...Song) PlaylistOption {
	return func(p *Playlist) {
		for _, s := range songs {
			_ = p.AddSong(s)
		}
	}
}

// PlayerOption is the functional option type used to configure a Player.
type PlayerOption func(*Player)

// WithRepeatMode sets the initial repeat mode of the player.
func WithRepeatMode(mode RepeatMode) PlayerOption {
	return func(p *Player) {
		p.repeatMode = mode
	}
}

// WithRandSource injects a custom random number generator for shuffle operations.
// Dependency Injection: pass a seeded *rand.Rand in tests to get deterministic shuffle.
func WithRandSource(r *rand.Rand) PlayerOption {
	return func(p *Player) {
		p.rng = r
	}
}

// WithEventListener registers an event listener on the player (Observer Pattern).
func WithEventListener(l EventListener) PlayerOption {
	return func(p *Player) {
		p.Subscribe(l)
	}
}
