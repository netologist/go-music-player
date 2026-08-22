package musicplayer

import (
	"testing"
	"time"
)

func mustSong(t *testing.T, id, title string) Song {
	t.Helper()
	s, err := NewSong(id, title, "Test Artist", 3*time.Minute)
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	return s
}

// TestAddSong_BasicAppend: temel ekleme işleminin sırayı koruduğunu
// ve Len()'in doğru arttığını doğrular.
func TestAddSong_BasicAppend(t *testing.T) {
	p := NewPlaylist("test", false)
	s1 := mustSong(t, "1", "Song A")
	s2 := mustSong(t, "2", "Song B")

	if err := p.AddSong(s1); err != nil {
		t.Fatalf("AddSong(s1) hata verdi: %v", err)
	}
	if err := p.AddSong(s2); err != nil {
		t.Fatalf("AddSong(s2) hata verdi: %v", err)
	}
	if p.Len() != 2 {
		t.Fatalf("Len() = %d, beklenen 2", p.Len())
	}

	songs := p.Songs()
	if songs[0].ID != "1" || songs[1].ID != "2" {
		t.Fatalf("ekleme sırası korunmadı: %+v", songs)
	}
}

// TestAddSong_DedupEnabled_RejectsDuplicate: dedup açıkken aynı ID'nin
// tekrar eklenemediğini doğrular.
func TestAddSong_DedupEnabled_RejectsDuplicate(t *testing.T) {
	p := NewPlaylist("test", true)
	s1 := mustSong(t, "1", "Song A")

	if err := p.AddSong(s1); err != nil {
		t.Fatalf("ilk ekleme başarısız: %v", err)
	}
	if err := p.AddSong(s1); err != ErrDuplicateSong {
		t.Fatalf("ikinci ekleme ErrDuplicateSong dönmeliydi, döndü: %v", err)
	}
	if p.Len() != 1 {
		t.Fatalf("dedup sonrası Len() = %d, beklenen 1", p.Len())
	}
}

// TestAddSong_DedupDisabled_AllowsDuplicate: dedup kapalıyken aynı ID'nin
// iki kez eklenebildiğini doğrular (varsayılan davranış farkı).
func TestAddSong_DedupDisabled_AllowsDuplicate(t *testing.T) {
	p := NewPlaylist("test", false)
	s1 := mustSong(t, "1", "Song A")

	_ = p.AddSong(s1)
	if err := p.AddSong(s1); err != nil {
		t.Fatalf("dedup kapalıyken tekrar ekleme başarısız olmamalıydı: %v", err)
	}
	if p.Len() != 2 {
		t.Fatalf("Len() = %d, beklenen 2", p.Len())
	}
}

// TestRemoveSong_ByID: ID ile silmenin doğru elemanı kaldırdığını ve
// index map'inin tutarlı kaldığını (bir sonraki AddSong'un doğru
// çalışmasıyla dolaylı olarak) doğrular.
func TestRemoveSong_ByID(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))
	_ = p.AddSong(mustSong(t, "2", "B"))
	_ = p.AddSong(mustSong(t, "3", "C"))

	if err := p.RemoveSong("2"); err != nil {
		t.Fatalf("RemoveSong hata verdi: %v", err)
	}
	if p.Contains("2") {
		t.Fatalf("silinen şarkı hâlâ Contains() ile bulunuyor")
	}

	songs := p.Songs()
	if len(songs) != 2 || songs[0].ID != "1" || songs[1].ID != "3" {
		t.Fatalf("silme sonrası sıra bozuldu: %+v", songs)
	}
}

// TestRemoveSong_NotFound: olmayan bir ID silinmeye çalışılırsa
// ErrSongNotFound dönmeli (edge case).
func TestRemoveSong_NotFound(t *testing.T) {
	p := NewPlaylist("test", false)
	if err := p.RemoveSong("ghost"); err != ErrSongNotFound {
		t.Fatalf("beklenen ErrSongNotFound, alınan: %v", err)
	}
}

// TestRemoveAt_InvalidIndex: geçersiz index (negatif veya sınır dışı)
// ErrInvalidIndex dönmeli.
func TestRemoveAt_InvalidIndex(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))

	if err := p.RemoveAt(-1); err != ErrInvalidIndex {
		t.Fatalf("negatif index için ErrInvalidIndex bekleniyordu: %v", err)
	}
	if err := p.RemoveAt(5); err != ErrInvalidIndex {
		t.Fatalf("sınır dışı index için ErrInvalidIndex bekleniyordu: %v", err)
	}
}

// TestMoveSong_Reorder: bir şarkıyı ileri ve geri taşımanın, hem slice
// sırasını hem de index map'ini doğru güncellediğini doğrular.
func TestMoveSong_Reorder(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))
	_ = p.AddSong(mustSong(t, "2", "B"))
	_ = p.AddSong(mustSong(t, "3", "C"))
	_ = p.AddSong(mustSong(t, "4", "D"))

	// "A"yı (index 0) index 2'ye taşı: beklenen sonuç [B, C, A, D]
	if err := p.MoveSong("1", 2); err != nil {
		t.Fatalf("MoveSong hata verdi: %v", err)
	}
	songs := p.Songs()
	got := []string{songs[0].ID, songs[1].ID, songs[2].ID, songs[3].ID}
	want := []string{"2", "3", "1", "4"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("taşıma sonrası sıra yanlış: got=%v want=%v", got, want)
		}
	}

	// Map'in hâlâ tutarlı olduğunu, "1" ID'li şarkıyı tekrar taşıyarak doğrula.
	if err := p.MoveSong("1", 0); err != nil {
		t.Fatalf("ikinci MoveSong hata verdi (map bozulmuş olabilir): %v", err)
	}
	songs = p.Songs()
	if songs[0].ID != "1" {
		t.Fatalf("ikinci taşıma sonrası beklenen ilk eleman '1', bulunan: %s", songs[0].ID)
	}
}

// TestMoveSong_NotFound: olmayan bir ID taşınmaya çalışılırsa hata dönmeli.
func TestMoveSong_NotFound(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))
	if err := p.MoveSong("ghost", 0); err != ErrSongNotFound {
		t.Fatalf("beklenen ErrSongNotFound, alınan: %v", err)
	}
}

// TestMoveSong_InvalidPosition: geçersiz hedef pozisyon ErrInvalidPosition
// dönmeli (edge case: negatif ya da sınır dışı).
func TestMoveSong_InvalidPosition(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))
	if err := p.MoveSong("1", 10); err != ErrInvalidPosition {
		t.Fatalf("beklenen ErrInvalidPosition, alınan: %v", err)
	}
}

// TestSongs_ReturnsCopyNotReference: Songs()'ın döndürdüğü slice'ta yapılan
// değişikliğin internal state'i ETKİLEMEDİĞİNİ doğrular. Bu, "neden kopya
// döndürüyoruz" tasarım kararının doğruluğunu kanıtlayan kritik bir test.
func TestSongs_ReturnsCopyNotReference(t *testing.T) {
	p := NewPlaylist("test", false)
	_ = p.AddSong(mustSong(t, "1", "A"))

	songs := p.Songs()
	songs[0].Title = "HACKED"

	internal := p.Songs()
	if internal[0].Title == "HACKED" {
		t.Fatalf("Songs() kopya değil referans dönmüş, internal state dışarıdan değiştirilebiliyor")
	}
}

// TestEmptyPlaylist_Operations: boş playlist üzerinde işlemlerin panik
// atmadan, anlamlı hatalarla döndüğünü doğrular (edge case).
func TestEmptyPlaylist_Operations(t *testing.T) {
	p := NewPlaylist("empty", false)

	if p.Len() != 0 {
		t.Fatalf("boş playlist Len() 0 olmalı")
	}
	if err := p.RemoveSong("x"); err != ErrSongNotFound {
		t.Fatalf("boş playlist'te silme ErrSongNotFound dönmeli: %v", err)
	}
	if _, err := p.At(0); err != ErrInvalidIndex {
		t.Fatalf("boş playlist'te At(0) ErrInvalidIndex dönmeli: %v", err)
	}
}
