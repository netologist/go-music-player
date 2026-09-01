package musicplayer

import (
	"encoding/json"
	"os"
)

// playlistSnapshot is the serialisable "flat" representation of a Playlist.
//
// Why not json.Marshal the Playlist directly: the internal state fields
// (mu sync.RWMutex, indexByID map) must not be serialised — a mutex cannot be
// marshalled, and indexByID is DERIVED state (it can be fully reconstructed from
// the songs slice), so writing it to disk would be redundant and risks
// desync on load. Only the true source fields (Name, Songs, DedupEnabled) are
// persisted; indexByID is rebuilt via AddSong calls on load.
type playlistSnapshot struct {
	Name         string `json:"name"`
	Songs        []Song `json:"songs"`
	DedupEnabled bool   `json:"dedup_enabled"`
}

// SaveToFile serialises the playlist to JSON on disk.
//
// Why JSON (and why encoding/json): the format is human-readable, the standard
// library is sufficient (no third-party dependency), and Song's fields are
// simple / flat, so a complex serialiser is unnecessary. For higher throughput
// (thousands of playlists, frequent saves) gob or protobuf would be worth
// considering, but at this scale JSON's readability advantage exceeds the
// performance difference.
//
// Concurrency note: RLock is acquired because we are only reading; other
// goroutines can continue READING the playlist during the save, but concurrent
// WRITE operations (Add/Remove/Move) are blocked.
func (p *Playlist) SaveToFile(path string) error {
	p.mu.RLock()
	snap := playlistSnapshot{
		Name:         p.Name,
		Songs:        append([]Song(nil), p.songs...),
		DedupEnabled: p.DedupEnabled,
	}
	p.mu.RUnlock()

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// LoadPlaylistFromFile reads a playlist from disk and REBUILDS the indexByID map
// via AddSong calls (the "don't persist derived state" problem described above).
func LoadPlaylistFromFile(path string) (*Playlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var snap playlistSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}

	// Dedup is disabled during loading so that songs already on disk
	// (possibly saved when dedup was off) don't silently disappear;
	// the actual DedupEnabled value is set at the very end.
	pl := NewPlaylist(snap.Name)
	for _, s := range snap.Songs {
		if err := pl.AddSong(s); err != nil {
			return nil, err
		}
	}
	pl.DedupEnabled = snap.DedupEnabled
	return pl, nil
}
