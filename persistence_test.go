package musicplayer

import (
	"path/filepath"
	"testing"
)

// TestSaveAndLoad_RoundTrip: bir playlist'i diske kaydedip geri yüklemenin,
// içeriği (şarkı sırası, ID'ler, dedup ayarı) BİREBİR koruduğunu doğrular.
func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir() // her test için izole, otomatik temizlenen geçici klasör
	path := filepath.Join(dir, "playlist.json")

	original := NewPlaylist("My Mix", true)
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
