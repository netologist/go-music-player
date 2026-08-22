package musicplayer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// TestSaveAndLoad_RoundTrip: bir playlist'i diske kaydedip geri yüklemenin,
// içeriği (şarkı sırası, ID'ler, dedup ayarı) BİREBİR koruduğunu doğrular.
func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir() // her test için izole, otomatik temizlenen geçici klasör
	path := filepath.Join(dir, "playlist.json")

	original := NewPlaylist("My Mix", WithDedup(true))
	_ = original.AddSong(mustSong(t, "1", "Song A"))
	_ = original.AddSong(mustSong(t, "2", "Song B"))
	_ = original.AddSong(mustSong(t, "3", "Song C"))

	if err := original.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile hata verdi: %v", err)
	}

	loaded, err := LoadPlaylistFromFile(path)
	if err != nil {
		t.Fatalf("LoadPlaylistFromFile hata verdi: %v", err)
	}

	if loaded.Name != original.Name {
		t.Fatalf("Name uyuşmuyor: got=%s want=%s", loaded.Name, original.Name)
	}
	if loaded.DedupEnabled != original.DedupEnabled {
		t.Fatalf("DedupEnabled uyuşmuyor: got=%v want=%v", loaded.DedupEnabled, original.DedupEnabled)
	}
	if loaded.Len() != original.Len() {
		t.Fatalf("Len uyuşmuyor: got=%d want=%d", loaded.Len(), original.Len())
	}

	origSongs := original.Songs()
	loadedSongs := loaded.Songs()
	for i := range origSongs {
		if origSongs[i].ID != loadedSongs[i].ID || origSongs[i].Title != loadedSongs[i].Title {
			t.Fatalf("index %d'de şarkı uyuşmuyor: got=%+v want=%+v", i, loadedSongs[i], origSongs[i])
		}
	}

	// Yüklenen playlist'in indexByID map'inin GERÇEKTEN yeniden inşa
	// edildiğini doğrula: dedup açık olduğu için aynı ID tekrar
	// eklenemez olmalı.
	if err := loaded.AddSong(mustSong(t, "1", "Duplicate Attempt")); err != ErrDuplicateSong {
		t.Fatalf("yüklenen playlist'te indexByID doğru inşa edilmemiş, dedup çalışmadı: %v", err)
	}
}

// TestLoadPlaylistFromFile_MissingFile_ReturnsError: olmayan bir dosya
// okunmaya çalışılırsa panik atmadan hata dönmesini doğrular (edge case).
func TestLoadPlaylistFromFile_MissingFile_ReturnsError(t *testing.T) {
	_, err := LoadPlaylistFromFile("/nonexistent/path/playlist.json")
	if err == nil {
		t.Fatalf("olmayan dosya için hata beklenirdi, nil döndü")
	}
}

// TestJSONFileRepository_CRUD: JSONFileRepository ile CRUD ve List işlemlerini doğrular.
func TestJSONFileRepository_CRUD(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewJSONFileRepository(dir)
	if err != nil {
		t.Fatalf("NewJSONFileRepository hata: %v", err)
	}

	ctx := context.Background()
	p := NewPlaylist("Rock Classics", WithDedup(true))
	_ = p.AddSong(mustSong(t, "1", "Bohemian Rhapsody"))
	_ = p.AddSong(mustSong(t, "2", "Stairway to Heaven"))

	// Save
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("repo.Save hata: %v", err)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("repo.List hata: %v", err)
	}
	if len(list) != 1 || list[0] != "Rock_Classics" {
		t.Fatalf("beklenen list ['Rock_Classics'], bulunan: %v", list)
	}

	// Load
	loaded, err := repo.Load(ctx, "Rock Classics")
	if err != nil {
		t.Fatalf("repo.Load hata: %v", err)
	}
	if loaded.Len() != 2 || loaded.Name != "Rock Classics" {
		t.Fatalf("yüklenen playlist uyuşmuyor: %s, len=%d", loaded.Name, loaded.Len())
	}

	// Delete
	if err := repo.Delete(ctx, "Rock Classics"); err != nil {
		t.Fatalf("repo.Delete hata: %v", err)
	}

	// Load after delete -> ErrPlaylistNotFound
	_, err = repo.Load(ctx, "Rock Classics")
	if !errors.Is(err, ErrPlaylistNotFound) {
		t.Fatalf("silinen playlist için ErrPlaylistNotFound bekleniyordu: %v", err)
	}
}

// TestMemoryRepository_CRUD: MemoryRepository ile in-memory CRUD işlemlerini doğrular.
func TestMemoryRepository_CRUD(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	p := NewPlaylist("Jazz Hits")
	_ = p.AddSong(mustSong(t, "j1", "Take Five"))

	// Save & Load
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("repo.Save hata: %v", err)
	}

	loaded, err := repo.Load(ctx, "Jazz Hits")
	if err != nil {
		t.Fatalf("repo.Load hata: %v", err)
	}
	if loaded.Len() != 1 {
		t.Fatalf("yüklenen playlist 1 şarkı içermeli, bulunan: %d", loaded.Len())
	}

	// List
	names, err := repo.List(ctx)
	if err != nil || len(names) != 1 || names[0] != "Jazz Hits" {
		t.Fatalf("List hatalı: %v, err: %v", names, err)
	}

	// Delete
	if err := repo.Delete(ctx, "Jazz Hits"); err != nil {
		t.Fatalf("Delete hata: %v", err)
	}

	// NotFound check
	if _, err := repo.Load(ctx, "Jazz Hits"); !errors.Is(err, ErrPlaylistNotFound) {
		t.Fatalf("ErrPlaylistNotFound bekleniyordu: %v", err)
	}
}

// TestRepository_ContextCancellation: Context iptal edildiğinde repository
// metodlarının context hatasıyla döndüğünü doğrular.
func TestRepository_ContextCancellation(t *testing.T) {
	repo := NewMemoryRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Hemen iptal et

	p := NewPlaylist("Test")
	if err := repo.Save(ctx, p); !errors.Is(err, context.Canceled) {
		t.Fatalf("iptal edilmiş context ile context.Canceled bekleniyordu, alınan: %v", err)
	}
	if _, err := repo.Load(ctx, "Test"); !errors.Is(err, context.Canceled) {
		t.Fatalf("iptal edilmiş context ile context.Canceled bekleniyordu, alınan: %v", err)
	}
}

// TestRepository_EmptyPlaylistName: İsimsiz playlist kaydedilememeli.
func TestRepository_EmptyPlaylistName(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	p := NewPlaylist("")
	if err := repo.Save(ctx, p); !errors.Is(err, ErrEmptyPlaylistName) {
		t.Fatalf("ErrEmptyPlaylistName bekleniyordu, alınan: %v", err)
	}
}
