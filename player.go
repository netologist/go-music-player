package musicplayer

import (
	"errors"
	"iter"
	"math/rand/v2"
	"sync"
)

// PlaybackState represents the current playback state of the Player.
type PlaybackState int

const (
	StateStopped PlaybackState = iota
	StatePlaying
	StatePaused
)

// RepeatMode controls the behaviour when the end (or beginning) of the queue is reached.
type RepeatMode int

const (
	RepeatOff RepeatMode = iota // stop at the end
	RepeatOne                   // repeat the current track
	RepeatAll                   // wrap around to the beginning
)

var ErrEndOfPlaylist = errors.New("reached end of playlist (repeat off)")
var ErrNotPlaying = errors.New("player is not currently playing")

// Player manages the playback STATE over a Playlist.
//
// Design decision: Player holds its own "queue" copy of the playlist's song
// slice rather than a direct reference to it. Reason: shuffle must not corrupt
// the playlist's canonical order (the user may still view the playlist in its
// original order on another screen). Shuffle only affects the Player's playback
// order; RestoreOrder returns to the original playlist sequence.
//
// The current track is tracked by currentSongID (not index). Reason: after a
// shuffle the index of the same song changes; if we only stored the index,
// shuffle would make the wrong song appear as "current". ID-based tracking
// guarantees the correct song is found after any shuffle/reorder (see
// syncCurrentIndexLocked).
type Player struct {
	mu             sync.Mutex
	playlist       *Playlist
	queue          []Song
	originalOrder  []Song
	currentIndex   int
	currentSongID  string
	state          PlaybackState
	repeatMode     RepeatMode
	rng            *rand.Rand
	listeners      map[int]EventListener
	nextListenerID int
}

// NewPlayer builds a player by taking a snapshot of the playlist's current state.
// PlayerOption functions can configure repeatMode, a rand source, and event listeners.
func NewPlayer(playlist *Playlist, opts ...PlayerOption) *Player {
	songs := playlist.Songs()
	p := &Player{
		playlist:      playlist,
		queue:         songs,
		originalOrder: append([]Song(nil), songs...),
		currentIndex:  0,
		state:         StateStopped,
		repeatMode:    RepeatOff,
		listeners:     make(map[int]EventListener),
	}
	if len(songs) > 0 {
		p.currentSongID = songs[0].ID
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Refresh re-synchronises the player's queue when the underlying playlist has
// changed externally (songs added or removed). If the currently playing song
// still exists in the playlist its position is preserved; otherwise the player
// resets to index 0.
func (p *Player) Refresh() {
	p.mu.Lock()
	songs := p.playlist.Songs()
	p.queue = songs
	p.originalOrder = append([]Song(nil), songs...)
	p.syncCurrentIndexLocked()
	evt, cbs := p.snapshotEventLocked(EventQueueUpdated)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
}

// syncCurrentIndexLocked searches for currentSongID in the queue and updates
// currentIndex accordingly. Falls back to index 0 if the song is not found
// (e.g. it was deleted). O(n) — only called after shuffle or refresh, not on
// the hot playback path.
func (p *Player) syncCurrentIndexLocked() {
	if len(p.queue) == 0 {
		p.currentIndex = 0
		p.currentSongID = ""
		return
	}
	for i, s := range p.queue {
		if s.ID == p.currentSongID {
			p.currentIndex = i
			return
		}
	}
	p.currentIndex = 0
	p.currentSongID = p.queue[0].ID
}

// Play transitions state to Playing. Returns an error if the queue is empty.
func (p *Player) Play() error {
	p.mu.Lock()
	if len(p.queue) == 0 {
		p.mu.Unlock()
		return ErrEmptyPlaylist
	}
	p.state = StatePlaying
	evt, cbs := p.snapshotEventLocked(EventStateChanged)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
	return nil
}

// Pause is only meaningful when already in the Playing state.
func (p *Player) Pause() error {
	p.mu.Lock()
	if p.state != StatePlaying {
		p.mu.Unlock()
		return ErrNotPlaying
	}
	p.state = StatePaused
	evt, cbs := p.snapshotEventLocked(EventStateChanged)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
	return nil
}

// Next advances to the next track according to the current repeat mode.
//
// Assumption (console-based, no real timer): there is no separate "song ended"
// trigger in this system; advancement is simulated exclusively via Next().
// Therefore RepeatOne intentionally keeps the same track — in a real player
// this maps to "replay the same song when it finishes".
func (p *Player) Next() error {
	p.mu.Lock()
	if len(p.queue) == 0 {
		p.mu.Unlock()
		return ErrEmptyPlaylist
	}

	if p.repeatMode == RepeatOne {
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	}

	if p.currentIndex+1 < len(p.queue) {
		p.currentIndex++
		p.currentSongID = p.queue[p.currentIndex].ID
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	}

	// We are at the end of the queue.
	switch p.repeatMode {
	case RepeatAll:
		p.currentIndex = 0
		p.currentSongID = p.queue[0].ID
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	default: // RepeatOff
		p.mu.Unlock()
		return ErrEndOfPlaylist
	}
}

// Previous is the mirror image of Next. Wraps to the end with RepeatAll.
func (p *Player) Previous() error {
	p.mu.Lock()
	if len(p.queue) == 0 {
		p.mu.Unlock()
		return ErrEmptyPlaylist
	}

	if p.repeatMode == RepeatOne {
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	}

	if p.currentIndex-1 >= 0 {
		p.currentIndex--
		p.currentSongID = p.queue[p.currentIndex].ID
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	}

	switch p.repeatMode {
	case RepeatAll:
		p.currentIndex = len(p.queue) - 1
		p.currentSongID = p.queue[p.currentIndex].ID
		evt, cbs := p.snapshotEventLocked(EventTrackChanged)
		p.mu.Unlock()
		dispatchEvent(evt, cbs)
		return nil
	default:
		p.mu.Unlock()
		return ErrEndOfPlaylist
	}
}

// SetRepeatMode changes the current repeat mode.
func (p *Player) SetRepeatMode(mode RepeatMode) {
	p.mu.Lock()
	p.repeatMode = mode
	evt, cbs := p.snapshotEventLocked(EventRepeatModeChanged)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
}

// Shuffle randomises the queue using the Fisher-Yates algorithm.
//
// Why Fisher-Yates: O(n) time, O(1) extra space (in-place), and provably
// uniform — every permutation is equally likely. A naive "swap N random
// pairs" approach can produce biased results.
//
// Because the current track is tracked by ID, syncCurrentIndexLocked after
// the shuffle guarantees the "currently playing song" identity is preserved.
func (p *Player) Shuffle() {
	p.mu.Lock()
	for i := len(p.queue) - 1; i > 0; i-- {
		var j int
		if p.rng != nil {
			j = p.rng.IntN(i + 1)
		} else {
			j = rand.IntN(i + 1)
		}
		p.queue[i], p.queue[j] = p.queue[j], p.queue[i]
	}
	p.syncCurrentIndexLocked()
	evt, cbs := p.snapshotEventLocked(EventQueueUpdated)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
}

// RestoreOrder returns the queue to the original playlist order (pre-shuffle).
func (p *Player) RestoreOrder() {
	p.mu.Lock()
	p.queue = append([]Song(nil), p.originalOrder...)
	p.syncCurrentIndexLocked()
	evt, cbs := p.snapshotEventLocked(EventQueueUpdated)
	p.mu.Unlock()
	dispatchEvent(evt, cbs)
}

// CurrentSong returns the song currently marked as "playing".
func (p *Player) CurrentSong() (Song, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return Song{}, ErrEmptyPlaylist
	}
	return p.queue[p.currentIndex], nil
}

// State returns the current playback state.
func (p *Player) State() PlaybackState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// RepeatModeValue returns the current repeat mode (useful for testing/inspection).
func (p *Player) RepeatModeValue() RepeatMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.repeatMode
}

// QueueLen returns the current length of the playback queue.
func (p *Player) QueueLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.queue)
}

// QueueSongs returns a COPY of the current playback order (for inspection/UI/testing).
// Same rationale as Songs(): the caller must not be able to mutate the player's
// internal state by modifying the returned slice.
func (p *Player) QueueSongs() []Song {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Song, len(p.queue))
	copy(out, p.queue)
	return out
}

// All returns an iter.Seq2 for ranging over the player's current queue with index and Song.
// Runs under mutex protection; the lock is released safely on early exit (break).
// Zero extra allocations (0-alloc).
func (p *Player) All() iter.Seq2[int, Song] {
	return LockedSeq2(&p.mu, func() []Song { return p.queue })
}

// Values returns an iter.Seq for ranging over the player's current queue (Song only).
// Runs under mutex protection; the lock is released safely on early exit (break).
// Zero extra allocations (0-alloc).
func (p *Player) Values() iter.Seq[Song] {
	return LockedSeq(&p.mu, func() []Song { return p.queue })
}

// ForEach runs the given function for every song in the player's queue.
// The loop exits early if fn returns false (short-circuiting).
// Uses the generic ForEach under the lock; produces 0 extra allocations.
func (p *Player) ForEach(fn func(index int, song Song) bool) {
	ForEach(p.All(), fn)
}

// Filter returns a new Song slice containing only the queue entries that match the predicate.
// Uses the generic Filter under the lock.
func (p *Player) Filter(predicate func(Song) bool) []Song {
	return Filter(p.Values(), predicate)
}

// Subscribe registers an EventListener to receive Player events (Observer Pattern).
// The returned unsubscribe function removes the listener when called.
func (p *Player) Subscribe(l EventListener) (unsubscribe func()) {
	if l == nil {
		return func() {}
	}
	p.mu.Lock()
	if p.listeners == nil {
		p.listeners = make(map[int]EventListener)
	}
	id := p.nextListenerID
	p.nextListenerID++
	p.listeners[id] = l
	p.mu.Unlock()

	return func() {
		p.mu.Lock()
		delete(p.listeners, id)
		p.mu.Unlock()
	}
}

// snapshotEventLocked copies the current state and listener list while p.mu is held.
// Listeners are invoked AFTER the lock is released to prevent deadlocks.
func (p *Player) snapshotEventLocked(eventType EventType) (PlayerEvent, []EventListener) {
	var cur Song
	if len(p.queue) > 0 && p.currentIndex < len(p.queue) {
		cur = p.queue[p.currentIndex]
	}
	evt := PlayerEvent{
		Type:         eventType,
		State:        p.state,
		CurrentSong:  cur,
		CurrentIndex: p.currentIndex,
		RepeatMode:   p.repeatMode,
		QueueLength:  len(p.queue),
	}
	cbs := make([]EventListener, 0, len(p.listeners))
	for _, l := range p.listeners {
		cbs = append(cbs, l)
	}
	return evt, cbs
}

func dispatchEvent(evt PlayerEvent, cbs []EventListener) {
	for _, cb := range cbs {
		cb(evt)
	}
}
