package musicplayer

import (
	"testing"
	"time"
)

// TestSongIterator_BasicOperations: HasNext, Next, Peek, Reset, Total, Remaining metodlarını doğrular.
func TestSongIterator_BasicOperations(t *testing.T) {
	s1, _ := NewSong("1", "Song 1", "Artist 1", 3*time.Minute)
	s2, _ := NewSong("2", "Song 2", "Artist 2", 4*time.Minute)
	s3, _ := NewSong("3", "Song 3", "Artist 3", 5*time.Minute)

	it := NewSongIterator([]Song{s1, s2, s3})

	if it.Total() != 3 {
		t.Fatalf("Total = %d, beklenen 3", it.Total())
	}
	if it.Remaining() != 3 {
		t.Fatalf("Remaining = %d, beklenen 3", it.Remaining())
	}

	// Peek
	peekSong, ok := it.Peek()
	if !ok || peekSong.ID != "1" {
		t.Fatalf("Peek() başarısız: %v, %v", peekSong, ok)
	}
	if it.Remaining() != 3 {
		t.Fatalf("Peek sonrası Remaining değişmemeliydi: %d", it.Remaining())
	}

	// 1. Next
	s, ok := it.Next()
	if !ok || s.ID != "1" {
		t.Fatalf("ilk Next() beklenen '1', alınan: %v", s.ID)
	}
	if it.Remaining() != 2 {
		t.Fatalf("Remaining = %d, beklenen 2", it.Remaining())
	}

	// 2. Next
	s, ok = it.Next()
	if !ok || s.ID != "2" {
		t.Fatalf("ikinci Next() beklenen '2', alınan: %v", s.ID)
	}

	// 3. Next
	s, ok = it.Next()
	if !ok || s.ID != "3" {
		t.Fatalf("üçüncü Next() beklenen '3', alınan: %v", s.ID)
	}

	if it.HasNext() {
		t.Fatalf("tüm elemanlar tüketildiğinde HasNext() false olmalı")
	}

	// Fazladan Next -> false
	_, ok = it.Next()
	if ok {
		t.Fatalf("tükenmiş iteratorda Next() ok=false dönmeli")
	}

	// Reset
	it.Reset()
	if !it.HasNext() || it.Remaining() != 3 {
		t.Fatalf("Reset sonrası iterator başa dönmeliydi")
	}
	s, ok = it.Next()
	if !ok || s.ID != "1" {
		t.Fatalf("Reset sonrası ilk eleman '1' olmalıydı: %v", s.ID)
	}
}

// TestPlaylist_Iterator: Playlist.Iterator() üzerinden gezinmeyi doğrular.
func TestPlaylist_Iterator(t *testing.T) {
	p := NewPlaylist("Rock")
	_ = p.AddSong(mustSong(t, "1", "Song A"))
	_ = p.AddSong(mustSong(t, "2", "Song B"))

	it := p.Iterator()
	var collected []string
	for it.HasNext() {
		s, ok := it.Next()
		if ok {
			collected = append(collected, s.ID)
		}
	}

	if len(collected) != 2 || collected[0] != "1" || collected[1] != "2" {
		t.Fatalf("Playlist.Iterator() eksik veya hatalı eleman döndü: %v", collected)
	}
}

// TestPlaylist_ForEach_EarlyExit: ForEach fonksiyonunun erken çıkışı (short-circuit) desteklediğini doğrular.
func TestPlaylist_ForEach_EarlyExit(t *testing.T) {
	p := NewPlaylist("Rock")
	_ = p.AddSong(mustSong(t, "1", "Song A"))
	_ = p.AddSong(mustSong(t, "2", "Song B"))
	_ = p.AddSong(mustSong(t, "3", "Song C"))

	count := 0
	p.ForEach(func(idx int, song Song) bool {
		count++
		// 2. elemandan sonra dur
		return idx < 1
	})

	if count != 2 {
		t.Fatalf("ForEach erken çıkışta 2 kez çalışmalıydı, çalışan: %d", count)
	}
}

// TestPlaylist_Filter: Playlist.Filter() fonksiyonunu doğrular.
func TestPlaylist_Filter(t *testing.T) {
	p := NewPlaylist("Rock")
	s1, _ := NewSong("1", "Queen - Song", "Queen", 3*time.Minute)
	s2, _ := NewSong("2", "ACDC - Song", "AC/DC", 4*time.Minute)
	s3, _ := NewSong("3", "Queen - Another", "Queen", 5*time.Minute)
	_ = p.AddSong(s1)
	_ = p.AddSong(s2)
	_ = p.AddSong(s3)

	queens := p.Filter(func(s Song) bool {
		return s.Artist == "Queen"
	})

	if len(queens) != 2 {
		t.Fatalf("Filter 2 Queen şarkısı bulmalıydı, bulunan: %d", len(queens))
	}
	if queens[0].ID != "1" || queens[1].ID != "3" {
		t.Fatalf("Filter yanlış şarkıları döndü: %+v", queens)
	}
}

// TestPlayer_QueueIterator: Player.QueueIterator() üzerinden queue gezinmesini doğrular.
func TestPlayer_QueueIterator(t *testing.T) {
	pl := NewPlaylist("Mix")
	_ = pl.AddSong(mustSong(t, "1", "A"))
	_ = pl.AddSong(mustSong(t, "2", "B"))

	player := NewPlayer(pl)
	it := player.QueueIterator()

	if it.Total() != 2 {
		t.Fatalf("QueueIterator Total = %d, beklenen 2", it.Total())
	}
	s, ok := it.Next()
	if !ok || s.ID != "1" {
		t.Fatalf("QueueIterator ilk eleman '1' olmalıydı")
	}
}
