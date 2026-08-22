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
	// ErrPlaylistNotFound, repository'de aranan playlist bulunamadığında döner.
	ErrPlaylistNotFound = errors.New("playlist repository'de bulunamadı")
	// ErrEmptyPlaylistName, isimsiz bir playlist kaydedilmeye çalışıldığında döner.
	ErrEmptyPlaylistName = errors.New("playlist adı boş olamaz")
)

// PlaylistRepository, Playlist'lerin saklanması, getirilmesi ve listelenmesini
// soyutlayan Repository Pattern arayüzüdür.
//
// Dependency Inversion Principle (DIP):
// Yüksek seviyeli müzik çalar mantığı doğrudan os.WriteFile veya veritabanı
// implementasyonlarına değil, bu repository arayüzüne bağımlıdır.
// Bu sayede disk (JSONFileRepository), hafıza (MemoryRepository) veya
// ileride SQL/NoSQL veritabanı adaptörleri sisteme sıfır kod değişikliğiyle eklenebilir.
type PlaylistRepository interface {
	Save(ctx context.Context, p *Playlist) error
	Load(ctx context.Context, name string) (*Playlist, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]string, error)
}

// JSONFileRepository, PlaylistRepository arayüzünü JSON dosyaları ve dosya sistemi
// üzerinde uygulayan somut bir Adapter'dır.
type JSONFileRepository struct {
	baseDir string
	mu      sync.RWMutex
}

// NewJSONFileRepository, belirtilen dizinde JSON tabanlı repository oluşturur.
// Dizin yoksa oluşturulur.
func NewJSONFileRepository(baseDir string) (*JSONFileRepository, error) {
	if baseDir == "" {
		baseDir = "."
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("repository dizini oluşturulamadı: %w", err)
	}
	return &JSONFileRepository{
		baseDir: baseDir,
	}, nil
}

func (r *JSONFileRepository) filePath(name string) string {
	cleanName := sanitizeFilename(name)
	return filepath.Join(r.baseDir, cleanName+".json")
}

// Save, playlist'i atomic write pratiği ile diske kaydeder (önce temp dosya, sonra rename).
// Bu pratik, yazma sırasında sistem çökmesi/kesinti durumunda dosyanın bozulmasını (corruption) önler.
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
		return fmt.Errorf("json marshal hatası: %w", err)
	}

	targetPath := r.filePath(p.Name)
	tmpPath := targetPath + ".tmp"

	// 1) Geçici dosyaya yaz
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("geçici dosya yazma hatası: %w", err)
	}

	// 2) Atomik rename ile hedef dosyanın üzerine yaz
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomik kaydetme hatası: %w", err)
	}

	return nil
}

// Load, verilen isimdeki playlist'i diskten okur ve yeniden inşa eder.
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
		return nil, fmt.Errorf("dosya okuma hatası: %w", err)
	}

	var snap playlistSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("json unmarshal hatası: %w", err)
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

// Delete, belirtilen playlist dosyasını siler.
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

// List, repository dizinindeki tüm playlist isimlerini döner.
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
		return nil, fmt.Errorf("dizin listeleme hatası: %w", err)
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

// MemoryRepository, testler ve geçici saklama için thread-safe in-memory
// PlaylistRepository adapter'ıdır.
type MemoryRepository struct {
	mu        sync.RWMutex
	playlists map[string]playlistSnapshot
}

// NewMemoryRepository, yeni bir MemoryRepository örneği oluşturur.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		playlists: make(map[string]playlistSnapshot),
	}
}

// Save, playlist'in bir kopyasını hafızada saklar.
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

// Load, hafızadan playlist'i yükler.
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

// Delete, playlist'i hafızadan kaldırır.
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

// List, hafızadaki tüm playlist isimlerini döner.
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
