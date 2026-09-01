package musicplayer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	// ErrPlaylistNotFound is returned when the requested playlist is not found in the repository.
	ErrPlaylistNotFound = errors.New("playlist not found in repository")
	// ErrEmptyPlaylistName is returned when attempting to save a playlist with no name.
	ErrEmptyPlaylistName = errors.New("playlist name cannot be empty")
)

// PlaylistRepository is the Repository Pattern interface that abstracts storing,
// retrieving, and listing Playlists.
//
// Dependency Inversion Principle (DIP):
// High-level music-player logic depends on this interface rather than directly
// on os.WriteFile or a specific database implementation. This means disk
// (JSONFileRepository), in-memory (MemoryRepository), or future SQL/NoSQL
// adapters can be swapped in without changing any existing code.
type PlaylistRepository interface {
	Save(ctx context.Context, p *Playlist) error
	Load(ctx context.Context, name string) (*Playlist, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]string, error)
}

// JSONFileRepository is a concrete Adapter that implements PlaylistRepository
// on top of JSON files and the file system.
type JSONFileRepository struct {
	baseDir string
	mu      sync.RWMutex
}

// NewJSONFileRepository creates a JSON-based repository in the given directory.
// The directory is created if it does not yet exist.
func NewJSONFileRepository(baseDir string) (*JSONFileRepository, error) {
	if baseDir == "" {
		baseDir = "."
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create repository directory: %w", err)
	}
	return &JSONFileRepository{
		baseDir: baseDir,
	}, nil
}

func (r *JSONFileRepository) filePath(name string) string {
	cleanName := sanitizeFilename(name)
	return filepath.Join(r.baseDir, cleanName+".json")
}

// Save writes the playlist to disk using the atomic write pattern (write to a
// temp file then rename). This prevents file corruption if the process crashes
// or is interrupted during the write.
func (r *JSONFileRepository) Save(ctx context.Context, p *Playlist) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if p == nil || strings.TrimSpace(p.Name) == "" {
		return ErrEmptyPlaylistName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	p.mu.RLock()
	snap := playlistSnapshot{
		Name:         p.Name,
		Songs:        append([]Song(nil), p.songs...),
		DedupEnabled: p.DedupEnabled,
	}
	p.mu.RUnlock()

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	targetPath := r.filePath(p.Name)
	tmpPath := targetPath + ".tmp"

	// 1) Write to a temporary file.
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 2) Atomic rename over the target file.
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomic save failed: %w", err)
	}

	return nil
}

// Load reads the named playlist from disk and rebuilds it.
func (r *JSONFileRepository) Load(ctx context.Context, name string) (*Playlist, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	path := r.filePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("file read error: %w", err)
	}

	var snap playlistSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	pl := NewPlaylist(snap.Name)
	for _, s := range snap.Songs {
		if err := pl.AddSong(s); err != nil {
			return nil, err
		}
	}
	pl.DedupEnabled = snap.DedupEnabled
	return pl, nil
}

// Delete removes the named playlist file from disk.
func (r *JSONFileRepository) Delete(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	path := r.filePath(name)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrPlaylistNotFound
		}
		return err
	}
	return nil
}

// List returns the names of all playlists in the repository directory.
func (r *JSONFileRepository) List(ctx context.Context) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		return nil, fmt.Errorf("directory listing error: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			names = append(names, name)
		}
	}
	return names, nil
}

// MemoryRepository is a thread-safe in-memory PlaylistRepository adapter
// intended for tests and temporary storage.
type MemoryRepository struct {
	mu        sync.RWMutex
	playlists map[string]playlistSnapshot
}

// NewMemoryRepository creates a new MemoryRepository instance.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		playlists: make(map[string]playlistSnapshot),
	}
}

// Save stores a copy of the playlist in memory.
func (m *MemoryRepository) Save(ctx context.Context, p *Playlist) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if p == nil || strings.TrimSpace(p.Name) == "" {
		return ErrEmptyPlaylistName
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	p.mu.RLock()
	snap := playlistSnapshot{
		Name:         p.Name,
		Songs:        append([]Song(nil), p.songs...),
		DedupEnabled: p.DedupEnabled,
	}
	p.mu.RUnlock()

	m.playlists[p.Name] = snap
	return nil
}

// Load retrieves a playlist from memory.
func (m *MemoryRepository) Load(ctx context.Context, name string) (*Playlist, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	snap, exists := m.playlists[name]
	if !exists {
		return nil, ErrPlaylistNotFound
	}

	pl := NewPlaylist(snap.Name)
	for _, s := range snap.Songs {
		if err := pl.AddSong(s); err != nil {
			return nil, err
		}
	}
	pl.DedupEnabled = snap.DedupEnabled
	return pl, nil
}

// Delete removes a playlist from memory.
func (m *MemoryRepository) Delete(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.playlists[name]; !exists {
		return ErrPlaylistNotFound
	}
	delete(m.playlists, name)
	return nil
}

// List returns all playlist names stored in memory.
func (m *MemoryRepository) List(ctx context.Context) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.playlists))
	for name := range m.playlists {
		names = append(names, name)
	}
	return names, nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}
