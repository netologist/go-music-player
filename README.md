# go-music-player

A fully-featured, production-quality music player library written in Go. This project is a **design patterns tutorial** disguised as a real application — every design decision is documented inline, and the README walks you through each layer from the ground up.

> **Go version:** 1.23+ (uses `iter.Seq` / `iter.Seq2` range-over-func, introduced in Go 1.23)

---

## Table of Contents

1. [What this project is](#what-this-project-is)
2. [Architecture overview](#architecture-overview)
3. [File-by-file walkthrough](#file-by-file-walkthrough)
   - [song.go — the domain entity](#songgo--the-domain-entity)
   - [errors.go — sentinel errors](#errorsgo--sentinel-errors)
   - [options.go — Functional Options Pattern](#optionsgo--functional-options-pattern)
   - [playlist.go — Playlist with dual data structure](#playlistgo--playlist-with-dual-data-structure)
   - [player.go — stateful playback engine](#playergo--stateful-playback-engine)
   - [builder.go — Builder Pattern](#buildergo--builder-pattern)
   - [iterator.go — generic iterators (Go 1.23 range-over-func)](#iteratongo--generic-iterators-go-123-range-over-func)
   - [events.go — Observer Pattern](#eventsgo--observer-pattern)
   - [merge.go — Strategy Pattern](#mergego--strategy-pattern)
   - [persistence.go — atomic JSON serialisation](#persistencego--atomic-json-serialisation)
   - [repository.go — Repository Pattern & DIP](#repositorygo--repository-pattern--dip)
4. [Design patterns used](#design-patterns-used)
5. [Key data-structure decisions](#key-data-structure-decisions)
6. [Concurrency model](#concurrency-model)
7. [Running the demo](#running-the-demo)
8. [Running the tests](#running-the-tests)
9. [Extending the project](#extending-the-project)

---

## What this project is

This is not a real audio player — there is no sound output. It is a **well-architected domain model** that simulates the core concerns of a music player:

| Concern | Solution |
|---|---|
| Representing songs and playlists | Domain model (`Song`, `Playlist`) |
| Flexible object construction | Builder Pattern + Functional Options |
| Playback state machine | `Player` with `Play`, `Pause`, `Next`, `Previous`, `Shuffle` |
| Reacting to state changes | Observer Pattern via `EventListener` |
| Merging two playlists | Strategy Pattern (`KeepFirst`, `KeepLast`, `KeepBoth`, `KeepLongest`) |
| Persisting to disk | Repository Pattern with atomic JSON writes |
| Efficient iteration | Go 1.23 `iter.Seq` / `iter.Seq2` with zero extra allocations |
| Thread safety | `sync.Mutex` / `sync.RWMutex` everywhere |

---

## Architecture overview

```
musicplayer/
├── song.go          # Domain entity: Song struct + constructor
├── errors.go        # Sentinel errors (errors.Is-compatible)
├── events.go        # Observer types: EventType, PlayerEvent, EventListener
├── options.go       # Functional Options for Playlist and Player
├── playlist.go      # Playlist: slice + O(1) map index, thread-safe
├── player.go        # Player: playback state machine, shuffle, queue, subscribe
├── builder.go       # Builder Pattern: SongBuilder, PlaylistBuilder (fluent API)
├── iterator.go      # Generic iterators: LockedSeq2, Filter, Map, Find, Collect …
├── merge.go         # Strategy Pattern: MergeStrategy interface + 4 strategies
├── persistence.go   # Atomic JSON save/load (low-level, no context)
├── repository.go    # Repository Pattern: PlaylistRepository, JSONFileRepository, MemoryRepository
└── cmd/
    └── demo/
        └── main.go  # Runnable demo wiring all components together
```

---

## File-by-file walkthrough

### `song.go` — the domain entity

```go
type Song struct {
    ID       string
    Title    string
    Artist   string
    Duration time.Duration
}
```

`Song` is a **value type** (struct, not pointer). It is intentionally simple and immutable by convention — nobody should mutate a song in-place after it is created.

**Why a string ID?** Every higher-level operation (dedup, merge conflict resolution, current-track tracking) needs a stable, unique key. A UUID or catalog ID fits better than a positional index because indices change when you shuffle or remove songs. The `NewSong` constructor enforces a non-empty ID as the single invariant.

---

### `errors.go` — sentinel errors

```go
var (
    ErrEmptySongID     = errors.New("song id cannot be empty")
    ErrSongNotFound    = errors.New("song not found")
    ErrDuplicateSong   = errors.New("a song with this id already exists in the playlist")
    ErrInvalidIndex    = errors.New("invalid index")
    ErrEmptyPlaylist   = errors.New("playlist is empty")
    ErrInvalidPosition = errors.New("invalid position")
)
```

Go's idiomatic approach to error handling: **sentinel errors** let callers distinguish failure kinds with `errors.Is`:

```go
err := playlist.AddSong(song)
if errors.Is(err, musicplayer.ErrDuplicateSong) {
    // handle duplicate specifically
}
```

This is strictly better than comparing error strings or using custom types for simple cases.

---

### `options.go` — Functional Options Pattern

**Problem:** How do you configure a struct with optional parameters without:
- A constructor with 10 arguments (brittle, hard to read)
- Multiple `New` functions (`NewPlayerWithRepeat`, `NewPlayerWithShuffle`, …)
- The "boolean trap" (`NewPlayer(playlist, true, false, nil)` — what does `true` mean?)

**Solution:** Functional Options Pattern.

```go
type PlayerOption func(*Player)

func WithRepeatMode(mode RepeatMode) PlayerOption {
    return func(p *Player) { p.repeatMode = mode }
}

func WithRandSource(r *rand.Rand) PlayerOption {
    return func(p *Player) { p.rng = r }
}

// Usage — self-documenting at the call site:
player := NewPlayer(playlist,
    WithRepeatMode(RepeatAll),
    WithRandSource(seededRng),
    WithEventListener(myHandler),
)
```

Each option is a closure that mutates the struct during construction. Adding a new option never changes existing call sites.

---

### `playlist.go` — Playlist with dual data structure

`Playlist` uses **two data structures working together**:

```go
type Playlist struct {
    mu           sync.RWMutex
    Name         string
    songs        []Song        // the ordered collection
    indexByID    map[string]int // songID -> current index in songs
    DedupEnabled bool
}
```

#### Why a slice, not a linked list?

| Operation | Slice | Linked list |
|---|---|---|
| Append | O(1) amortized | O(1) |
| Access by index | O(1) | O(n) |
| Remove by position | O(n) shift | O(1) — but only if you hold the node |
| Memory layout | Contiguous (cache-friendly) | Scattered (pointer chasing) |

The key insight: removing a song requires **finding it first** (we only have an ID, not a node pointer). Finding a node in a linked list is O(n). So the linked list's O(1) removal advantage evaporates — we would still be O(n) end-to-end.

A slice is more cache-friendly and idiomatic in Go. We accept O(n) remove/move in exchange for simpler, faster code everywhere else.

#### Why the `indexByID` map?

Without it, `Contains(id)` and `RemoveSong(id)` would both scan the entire slice — O(n). With the map, they are O(1):

```go
func (p *Playlist) Contains(id string) bool {
    p.mu.RLock()
    defer p.mu.RUnlock()
    _, ok := p.indexByID[id]
    return ok
}
```

The cost: after every `Remove` or `Move`, we call `reindexFrom(i)` to update all map entries from position `i` onwards. This is O(n) but runs only when the structure changes, not on every read.

#### Why does `Songs()` return a copy?

```go
func (p *Playlist) Songs() []Song {
    p.mu.RLock()
    defer p.mu.RUnlock()
    out := make([]Song, len(p.songs))
    copy(out, p.songs)
    return out
}
```

If we returned `p.songs` directly, the caller could `append` to it or assign to indices outside the lock — corrupting internal state or causing a data race. Returning a copy is a defensive guarantee: *"this is a snapshot; mutate it all you want."*

---

### `player.go` — stateful playback engine

`Player` is the most complex type. Key design decisions:

#### 1. Player owns its own queue (not the playlist slice)

```go
type Player struct {
    playlist      *Playlist // reference to original — used by Refresh()
    queue         []Song    // player's own copy (shuffle-safe)
    originalOrder []Song    // snapshot of the playlist at creation time
    currentIndex  int
    currentSongID string    // tracked by ID, not index
    ...
}
```

Shuffle must not corrupt the playlist's canonical order. A user viewing the playlist in one screen should still see it in order while the player shuffles it in another. Solution: **copy the songs into `queue`** at construction time. Shuffle only touches `queue`; `RestoreOrder()` copies `originalOrder` back.

#### 2. Track current song by ID, not by index

After a shuffle, the same song's index changes. If we only stored `currentIndex = 3`, after shuffling `queue[3]` would be a different song. Storing `currentSongID` means we can always find the right song regardless of what position it shuffled to:

```go
func (p *Player) syncCurrentIndexLocked() {
    for i, s := range p.queue {
        if s.ID == p.currentSongID {
            p.currentIndex = i
            return
        }
    }
    // Song was removed — fall back to start.
    p.currentIndex = 0
    p.currentSongID = p.queue[0].ID
}
```

#### 3. Fisher-Yates shuffle

```go
func (p *Player) Shuffle() {
    p.mu.Lock()
    for i := len(p.queue) - 1; i > 0; i-- {
        j := rand.IntN(i + 1)
        p.queue[i], p.queue[j] = p.queue[j], p.queue[i]
    }
    p.syncCurrentIndexLocked()
    ...
}
```

**Why Fisher-Yates specifically?**
- O(n) time, O(1) space (in-place)
- Provably uniform: every permutation has exactly equal probability
- A naive "pick two random positions and swap, repeat N times" approach produces biased output

#### 4. Deadlock-free event dispatch

Listeners must not be called while the mutex is held (a listener might call back into the player, causing a deadlock):

```go
// Lock is held here — snapshot state and listener list.
evt, cbs := p.snapshotEventLocked(EventStateChanged)
p.mu.Unlock() // Release BEFORE calling listeners.
dispatchEvent(evt, cbs) // Safe: no mutex held.
```

#### 5. RepeatOne intentionally stays on the same track for `Next()`

In a console simulation without a real timer, `Next()` is the only way to advance. In RepeatOne mode, `Next()` staying on the same song is correct — it mirrors "the song ends, and then the same song starts again" in a real player.

---

### `builder.go` — Builder Pattern

Two builders provide a **fluent API** for constructing domain objects:

```go
// SongBuilder
song := NewSongBuilder("s1").
    Title("Bohemian Rhapsody").
    Artist("Queen").
    Duration(6 * time.Minute).
    MustBuild()

// PlaylistBuilder
playlist := NewPlaylistBuilder("Rock Classics").
    WithDedup(true).
    AddNewSong("r1", "Bohemian Rhapsody", "Queen", 6*time.Minute).
    AddNewSong("r2", "Sweet Child O' Mine", "Guns N' Roses", 5*time.Minute).
    MustBuild()
```

Benefits:
- **Readable call sites**: each step is named
- **Accumulate errors lazily**: `AddNewSong` stores errors internally; `Build()` returns them all at once (no need to check an error after every single call)
- **`MustBuild()`**: panics on error — safe for test fixtures and `init`-style code where a panic is acceptable

---

### `iterator.go` — generic iterators (Go 1.23 range-over-func)

Go 1.23 introduced the `iter` package and the ability to range over functions directly:

```go
// Before Go 1.23 — had to materialise a slice:
for _, song := range playlist.Songs() { ... }

// Go 1.23 — zero extra allocation:
for i, song := range playlist.All() { ... }
```

Both `Playlist` and `Player` expose three iteration APIs:

| Method | Type | Usage |
|---|---|---|
| `All()` | `iter.Seq2[int, Song]` | `for i, s := range pl.All()` |
| `Values()` | `iter.Seq[Song]` | `for s := range pl.Values()` |
| `ForEach(fn)` | `func(int, Song) bool` | callback style with short-circuit |

The `LockedSeq2` / `LockedSeq` helpers acquire the lock for the duration of the iteration and release it on early exit (`break`), making all three APIs concurrency-safe at zero allocation cost.

Generic utility functions in `iterator.go`:

```go
// Lazy filter — no allocation until materialised.
active := Filter(playlist.Values(), func(s Song) bool {
    return s.Duration > 3*time.Minute
})

// Lazy transform.
titles := Map(playlist.Values(), func(s Song) string { return s.Title })

// First match.
song, ok := Find(playlist.Values(), func(s Song) bool { return s.ID == "r1" })
```

---

### `events.go` — Observer Pattern

`Player` implements the **Observer Pattern** to decouple internal state changes from their consumers (UI, logger, audio engine, analytics):

```go
type EventType int
const (
    EventStateChanged EventType = iota // Play / Pause
    EventTrackChanged                  // Next / Previous
    EventQueueUpdated                  // Shuffle / Refresh
    EventRepeatModeChanged             // SetRepeatMode
)

type PlayerEvent struct {
    Type         EventType
    State        PlaybackState
    CurrentSong  Song
    CurrentIndex int
    RepeatMode   RepeatMode
    QueueLength  int
}

type EventListener func(event PlayerEvent)
```

Registering a listener:

```go
unsubscribe := player.Subscribe(func(e musicplayer.PlayerEvent) {
    log.Printf("[%s] now playing: %s", e.Type, e.CurrentSong.Title)
})
defer unsubscribe()
```

Or inject at construction time:

```go
player := NewPlayer(playlist, WithEventListener(myHandler))
```

Listeners are stored in a `map[int]EventListener` (not a slice) so that `unsubscribe` can remove a listener in O(1) without shifting. Each listener gets a monotonically increasing integer key.

---

### `merge.go` — Strategy Pattern

**Problem:** Two playlists share some songs (same ID). How do you decide which version to keep when merging?

**Solution:** Strategy Pattern — the merge algorithm is behind an interface:

```go
type MergeStrategy interface {
    Merge(s1, s2 []Song) []Song
}
```

Four built-in strategies:

| Strategy | Conflict resolution |
|---|---|
| `KeepFirst` | Keep the version from the first playlist |
| `KeepLast` | Keep the version from the second playlist |
| `KeepBoth` | Keep both (song may appear twice) |
| `KeepLongest` | Keep the version with the longer `Duration` |

```go
merged := MergePlaylists(rock, indie, KeepFirst)
merged := MergePlaylists(rock, indie, KeepLongest)

// Or provide your own:
custom := MergeStrategyFunc(func(s1, s2 []Song) []Song {
    // your custom dedup/ordering logic
    return append(s1, s2...)
})
merged := MergePlaylists(rock, indie, custom)
```

Adding a new strategy requires zero changes to existing code (Open-Closed Principle).

---

### `persistence.go` — atomic JSON serialisation

`Playlist.SaveToFile` uses the **write-then-rename** pattern (atomic write):

```go
func (p *Playlist) SaveToFile(path string) error {
    // 1. Write to a temporary file.
    tmpPath := path + ".tmp"
    os.WriteFile(tmpPath, data, 0o644)

    // 2. Atomically rename over the target.
    os.Rename(tmpPath, path)
}
```

**Why atomic?** If the process crashes mid-write, the target file is left intact. A partial write only corrupts the `.tmp` file, which is discarded. This is the standard technique used by databases, editors, and package managers.

**Why not serialize `indexByID`?** It is derived state — fully reconstructible from the `songs` slice via `AddSong` calls during load. Serialising derived state wastes space and risks desync (what if the code changes how indexByID is computed?). Only the source of truth is persisted.

---

### `repository.go` — Repository Pattern & DIP

**Problem:** If the player's logic calls `os.WriteFile` directly, you cannot test it without a real file system, and you cannot swap to a database later without rewriting everything.

**Solution:** Repository Pattern with Dependency Inversion Principle.

```go
type PlaylistRepository interface {
    Save(ctx context.Context, p *Playlist) error
    Load(ctx context.Context, name string) (*Playlist, error)
    Delete(ctx context.Context, name string) error
    List(ctx context.Context) ([]string, error)
}
```

Two concrete implementations:

| Implementation | Use case |
|---|---|
| `JSONFileRepository` | Production — stores each playlist as a JSON file on disk |
| `MemoryRepository` | Tests and ephemeral storage — fully in-memory |

```go
// Production
repo, _ := NewJSONFileRepository("/var/lib/musicplayer")

// Tests — swap with zero code changes
repo := NewMemoryRepository()

// Both satisfy the same interface:
var repo PlaylistRepository = NewMemoryRepository()
_ = repo.Save(ctx, playlist)
```

All repository methods accept a `context.Context` so they can be cancelled cleanly (timeout, request cancellation).

---

## Design patterns used

| Pattern | Where | Why |
|---|---|---|
| **Builder** | `SongBuilder`, `PlaylistBuilder` | Fluent construction, lazy error accumulation, readable call sites |
| **Functional Options** | `PlaylistOption`, `PlayerOption` | Optional config without constructor explosion or boolean traps |
| **Observer** | `Player.Subscribe`, `EventListener` | Decouple state changes from UI/logging/audio without tight coupling |
| **Strategy** | `MergeStrategy` + 4 implementations | Swap merge algorithms at runtime; add new ones without touching existing code |
| **Repository** | `PlaylistRepository`, `JSONFileRepository`, `MemoryRepository` | Decouple persistence mechanism from domain logic; easy to test |
| **Adapter** | `MergeStrategyFunc` | Lets a plain function satisfy the `MergeStrategy` interface |

---

## Key data-structure decisions

### Slice over linked list for `Playlist.songs`

Real playlists: tens to thousands of songs, not millions. At that scale, slice wins on cache locality, simplicity, and idiomatic Go. The theoretical O(1) linked-list removal advantage requires a node reference — which we don't have (we search by ID). So both are O(n) end-to-end; the slice is just faster in practice.

### `indexByID map[string]int` alongside the slice

Gives O(1) existence checks and O(1) "find index by ID". The cost is O(n) re-indexing after a remove or move (`reindexFrom`), which is acceptable because structural mutations are infrequent compared to reads.

### Player tracks current song by ID, not index

Index-based tracking breaks after shuffle (the same song is now at a different position). ID-based tracking is invariant under reordering.

---

## Concurrency model

| Type | Lock type | Reason |
|---|---|---|
| `Playlist` | `sync.RWMutex` | Multiple readers (`Songs`, `At`, `Contains`, iteration) are allowed concurrently; writers (`AddSong`, `RemoveSong`, `MoveSong`) are exclusive |
| `Player` | `sync.Mutex` | All state mutations (play, pause, next, shuffle) are exclusive; no read-only methods complex enough to benefit from RW splitting |
| `MemoryRepository` | `sync.RWMutex` | Same rationale as `Playlist` |
| `JSONFileRepository` | `sync.RWMutex` | Reads (Load, List) can be concurrent; writes (Save, Delete) are exclusive |

**Deadlock prevention:** The Player takes its lock, snapshots the event and listener list, releases the lock, *then* calls listeners. This prevents a listener that calls back into the player from deadlocking.

---

## Running the demo

```bash
cd cmd/demo
go run .
```

Expected output (shuffle order may vary):

```
Rock playlist: [r1 r2 r3]
Indie playlist: [i1 r2 i2]
Merged (KeepFirst): [r1 r2 r3 i1 i2]
Merged songs (iter.Seq2): [r1: Bohemian Rhapsody] [r2: Sweet Child O' Mine] [r3: Back In Black] [i1: Midnight City] [i2: Feel Good Inc.]
  [Event: StateChanged] Song: Bohemian Rhapsody, State: 1
Now playing: Bohemian Rhapsody
  [Event: TrackChanged] Song: Sweet Child O' Mine, State: 1
After Next: Sweet Child O' Mine
  [Event: RepeatModeChanged] Song: Sweet Child O' Mine, State: 1
  [Event: QueueUpdated] Song: <shuffled>, State: 1
After Shuffle, now playing: <shuffled>
Playlist saved via Repository (JSONFileRepository).
Playlist loaded from repository: [r1 r2 r3 i1 i2]
```

---

## Running the tests

```bash
go test ./...
go test -race ./...          # with race detector
go test -cover ./...         # with coverage report
```

Tests cover:
- `playlist_test.go` — add, remove, move, dedup, concurrent access
- `player_test.go` — play, pause, next, previous, repeat modes, shuffle, restore order, events
- `builder_test.go` — fluent API, error accumulation, MustBuild panic
- `merge_test.go` — all four strategies, nil strategy fallback
- `iterator_test.go` — Filter, Map, Find, Collect, LockedSeq2, early exit
- `persistence_test.go` — save/load roundtrip, atomic write, derived state reconstruction
- `concurrency_test.go` — concurrent reads and writes under the race detector

---

## Extending the project

**Add a new merge strategy:**
```go
type byArtistStrategy struct{ preferred string }

func (s byArtistStrategy) Merge(s1, s2 []Song) []Song {
    // keep songs by preferred artist, fall back to s1
    ...
}

merged := MergePlaylists(rock, indie, byArtistStrategy{preferred: "Queen"})
```

**Add a new repository backend (e.g. SQLite):**
```go
type SQLiteRepository struct { db *sql.DB }

func (r *SQLiteRepository) Save(ctx context.Context, p *Playlist) error { ... }
func (r *SQLiteRepository) Load(ctx context.Context, name string) (*Playlist, error) { ... }
func (r *SQLiteRepository) Delete(ctx context.Context, name string) error { ... }
func (r *SQLiteRepository) List(ctx context.Context) ([]string, error) { ... }

// Zero changes to existing code — it satisfies PlaylistRepository.
var repo PlaylistRepository = &SQLiteRepository{db: db}
```

**Add a new event type:**
```go
const EventVolumeChanged EventType = 10 // add to events.go

// Fire it in player.go wherever volume is changed.
```

**Add a real audio engine:**
The `Player` is a pure state machine. Wire a real audio backend by subscribing to `EventTrackChanged` and starting/stopping audio playback in the listener.
