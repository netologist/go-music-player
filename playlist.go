package musicplayer

import (
	"iter"
	"sync"
)

// Playlist is an ordered collection of songs.
//
// Data structure decision (linked list vs slice):
//   - Slice was chosen because real-world playlists contain tens to thousands
//     of songs (not millions). At that scale, slice's cache-locality advantage
//     and its status as Go's idiomatic collection type outweigh the O(1)
//     mid-point removal advantage of a linked list.
//   - Remove/Move are O(n) on a slice (shift required); a linked list would make
//     them O(1) BUT only if you already hold a node reference (otherwise finding
//     the node is O(n) anyway). We search by song ID, so the linked list's
//     theoretical advantage disappears in practice.
//   - The indexByID map provides O(1) "does this ID exist?" / "what index is it at?"
//     lookups, which are critical for dedup checks and MoveSong(id, ...).
//     The cost: the map must be kept in sync after every Remove/Move
//     (see reindexFrom below).
type Playlist struct {
	mu           sync.RWMutex
	Name         string
	songs        []Song
	indexByID    map[string]int // songID -> current index in the songs slice
	DedupEnabled bool
}

// NewPlaylist creates a playlist. It can be configured flexibly via Functional
// Options Pattern — e.g. WithDedup or WithInitialSongs.
func NewPlaylist(name string, opts ...PlaylistOption) *Playlist {
	p := &Playlist{
		Name:         name,
		songs:        make([]Song, 0),
		indexByID:    make(map[string]int),
		DedupEnabled: false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// AddSong is O(1) amortized (append) + O(1) map insert.
// Returns ErrDuplicateSong if dedup is enabled and the ID already exists.
func (p *Playlist) AddSong(s Song) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.DedupEnabled {
		if _, exists := p.indexByID[s.ID]; exists {
			return ErrDuplicateSong
		}
	}

	p.songs = append(p.songs, s)
	p.indexByID[s.ID] = len(p.songs) - 1
	return nil
}

// RemoveSong removes a song by ID. O(n) because elements after the removed
// index must be shifted in the slice and their map entries updated.
func (p *Playlist) RemoveSong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx, exists := p.indexByID[id]
	if !exists {
		return ErrSongNotFound
	}
	return p.removeAtLocked(idx)
}

// RemoveAt removes a song by index (acquires the lock, then calls removeAtLocked).
func (p *Playlist) RemoveAt(index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < 0 || index >= len(p.songs) {
		return ErrInvalidIndex
	}
	return p.removeAtLocked(index)
}

// removeAtLocked assumes the mutex is ALREADY held (private helper).
// Why a separate "Locked" function: RemoveSong and RemoveAt share the same
// logic but are called with different inputs (id vs index); factoring the
// shared logic into a lock-free helper avoids double-locking (deadlock risk).
func (p *Playlist) removeAtLocked(index int) error {
	removedID := p.songs[index].ID
	p.songs = append(p.songs[:index], p.songs[index+1:]...)
	delete(p.indexByID, removedID)
	p.reindexFrom(index)
	return nil
}

// reindexFrom updates the map entries for all songs at index >= from after a
// remove or move. O(n) cost — the price for keeping O(1) dedup lookups.
func (p *Playlist) reindexFrom(from int) {
	for i := from; i < len(p.songs); i++ {
		p.indexByID[p.songs[i].ID] = i
	}
}

// MoveSong locates a song by ID and moves it to a new position (reorder).
// O(n) — elements between the old and new positions must shift.
func (p *Playlist) MoveSong(id string, newIndex int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldIndex, exists := p.indexByID[id]
	if !exists {
		return ErrSongNotFound
	}
	if newIndex < 0 || newIndex >= len(p.songs) {
		return ErrInvalidPosition
	}
	if oldIndex == newIndex {
		return nil
	}

	song := p.songs[oldIndex]
	// Remove from old position.
	p.songs = append(p.songs[:oldIndex], p.songs[oldIndex+1:]...)
	// Insert at new position.
	p.songs = append(p.songs, Song{})
	copy(p.songs[newIndex+1:], p.songs[newIndex:])
	p.songs[newIndex] = song

	// Everything between min(old,new) and max(old,new) may have shifted;
	// the safest approach is to reindex from the smaller of the two indices.
	from := oldIndex
	if newIndex < from {
		from = newIndex
	}
	p.reindexFrom(from)
	return nil
}

// Songs returns a COPY of the internal slice. Reason: if the caller holds the
// slice and mutates it outside the lock (append, index assignment), it would
// corrupt internal state or cause a data race. A copy prevents this.
func (p *Playlist) Songs() []Song {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]Song, len(p.songs))
	copy(out, p.songs)
	return out
}

// Len is O(1).
func (p *Playlist) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.songs)
}

// Contains is an O(1) map lookup.
func (p *Playlist) Contains(id string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.indexByID[id]
	return ok
}

// At returns a single song by index. O(1).
func (p *Playlist) At(index int) (Song, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if index < 0 || index >= len(p.songs) {
		return Song{}, ErrInvalidIndex
	}
	return p.songs[index], nil
}

// All returns an iter.Seq2 for ranging over the playlist with index and Song.
// Runs under RLock; the lock is released safely on early exit (break).
// Zero extra allocations (0-alloc).
func (p *Playlist) All() iter.Seq2[int, Song] {
	return LockedSeq2(p.mu.RLocker(), func() []Song { return p.songs })
}

// Values returns an iter.Seq for ranging over the playlist (Song only).
// Runs under RLock; the lock is released safely on early exit (break).
// Zero extra allocations (0-alloc).
func (p *Playlist) Values() iter.Seq[Song] {
	return LockedSeq(p.mu.RLocker(), func() []Song { return p.songs })
}

// ForEach runs the given function for every song in the playlist.
// The loop exits early if fn returns false (short-circuiting).
// Uses the generic ForEach under the lock; produces 0 extra allocations.
func (p *Playlist) ForEach(fn func(index int, song Song) bool) {
	ForEach(p.All(), fn)
}

// Filter returns a new Song slice containing only the songs that match the predicate.
// Uses the generic Filter under the lock.
func (p *Playlist) Filter(predicate func(Song) bool) []Song {
	return Filter(p.Values(), predicate)
}
