package musicplayer

import "testing"

func buildTestPlayer(t *testing.T) *Player {
	t.Helper()
	pl := NewPlaylist("test", false)
	_ = pl.AddSong(mustSong(t, "1", "A"))
	_ = pl.AddSong(mustSong(t, "2", "B"))
	_ = pl.AddSong(mustSong(t, "3", "C"))
	return NewPlayer(pl)
}

// TestPlay_EmptyPlaylist_ReturnsError: boş playlist ile Play() çağrısının
// panik atmadan anlamlı bir hata dönmesini doğrular.
func TestPlay_EmptyPlaylist_ReturnsError(t *testing.T) {
	pl := NewPlaylist("empty", false)
	p := NewPlayer(pl)

	if err := p.Play(); err != ErrEmptyPlaylist {
		t.Fatalf("boş playlist'te Play() ErrEmptyPlaylist dönmeli, alınan: %v", err)
	}
}

// TestPlayPauseStateTransitions: temel state machine geçişlerini doğrular.
func TestPlayPauseStateTransitions(t *testing.T) {
	p := buildTestPlayer(t)

	if p.State() != StateStopped {
		t.Fatalf("başlangıç state Stopped olmalı")
	}
	if err := p.Play(); err != nil {
		t.Fatalf("Play() hata verdi: %v", err)
	}
	if p.State() != StatePlaying {
		t.Fatalf("Play() sonrası state Playing olmalı")
	}
	if err := p.Pause(); err != nil {
		t.Fatalf("Pause() hata verdi: %v", err)
	}
	if p.State() != StatePaused {
		t.Fatalf("Pause() sonrası state Paused olmalı")
	}
}

// TestPause_WithoutPlaying_ReturnsError: çalmıyorken Pause() çağırmanın
// anlamlı bir hata döndüğünü doğrular (edge case).
func TestPause_WithoutPlaying_ReturnsError(t *testing.T) {
	p := buildTestPlayer(t)
	if err := p.Pause(); err != ErrNotPlaying {
		t.Fatalf("çalmıyorken Pause() ErrNotPlaying dönmeli, alınan: %v", err)
	}
}

// TestNext_RepeatOff_StopsAtEnd: RepeatOff modunda sona gelince
// ErrEndOfPlaylist dönmesini doğrular.
func TestNext_RepeatOff_StopsAtEnd(t *testing.T) {
	p := buildTestPlayer(t)
	p.SetRepeatMode(RepeatOff)

	_ = p.Next()                                  // 1 -> 2
	_ = p.Next()                                  // 2 -> 3
	if err := p.Next(); err != ErrEndOfPlaylist { // 3 -> son, hata bekleniyor
		t.Fatalf("RepeatOff'ta sonda ErrEndOfPlaylist bekleniyordu, alınan: %v", err)
	}
	cur, _ := p.CurrentSong()
	if cur.ID != "3" {
		t.Fatalf("hata sonrası hâlâ son şarkıda kalınmalı, bulunan: %s", cur.ID)
	}
}

// TestNext_RepeatAll_WrapsToStart: RepeatAll modunda sona gelince
// başa döndüğünü doğrular.
func TestNext_RepeatAll_WrapsToStart(t *testing.T) {
	p := buildTestPlayer(t)
	p.SetRepeatMode(RepeatAll)

	_ = p.Next() // 1 -> 2
	_ = p.Next() // 2 -> 3
	if err := p.Next(); err != nil {
		t.Fatalf("RepeatAll'da sonda hata dönmemeli: %v", err)
	}
	cur, _ := p.CurrentSong()
	if cur.ID != "1" {
		t.Fatalf("RepeatAll sonda başa (ID=1) dönmeliydi, bulunan: %s", cur.ID)
	}
}

// TestNext_RepeatOne_StaysOnSameSong: RepeatOne modunda Next() çağrısının
// şarkıyı DEĞİŞTİRMEDİĞİNİ doğrular (bkz. player.go'daki tasarım notu).
func TestNext_RepeatOne_StaysOnSameSong(t *testing.T) {
	p := buildTestPlayer(t)
	p.SetRepeatMode(RepeatOne)

	before, _ := p.CurrentSong()
	if err := p.Next(); err != nil {
		t.Fatalf("RepeatOne'da Next() hata dönmemeli: %v", err)
	}
	after, _ := p.CurrentSong()
	if before.ID != after.ID {
		t.Fatalf("RepeatOne'da şarkı değişmemeliydi: before=%s after=%s", before.ID, after.ID)
	}
}

// TestPrevious_RepeatOff_StopsAtStart: baştayken RepeatOff ile Previous()
// çağrısının hata dönmesini doğrular (Next testinin simetriği).
func TestPrevious_RepeatOff_StopsAtStart(t *testing.T) {
	p := buildTestPlayer(t)
	p.SetRepeatMode(RepeatOff)

	if err := p.Previous(); err != ErrEndOfPlaylist {
		t.Fatalf("baştayken RepeatOff'ta Previous() ErrEndOfPlaylist dönmeli, alınan: %v", err)
	}
}

// TestShuffle_PreservesSongCount: shuffle sonrası şarkı SAYISININ
// değişmediğini ve hiçbir şarkının kaybolmadığını (ID kümesi aynı)
// doğrular — bu, sorunun "shuffle sonrası eleman sayısı değişmemeli"
// gereksinimini doğrudan test eder.
//
// Not: Bunu Next() ile queue'yu gezerek DEĞİL, doğrudan QueueSongs() ile
// doğruluyoruz. Next() traversal'ı, shuffle sonrası currentIndex'in
// queue'nun ORTASINDA bir yerde olabileceği gerçeğiyle kırılgan olurdu
// (RepeatOff sadece ileri gider, başa dönmez) — bu yüzden en güvenilir
// doğrulama, queue'nun tamamını doğrudan okumaktır.
func TestShuffle_PreservesSongCount(t *testing.T) {
	p := buildTestPlayer(t)
	beforeIDs := map[string]bool{"1": true, "2": true, "3": true}

	p.Shuffle()

	after := p.QueueSongs()
	if len(after) != len(beforeIDs) {
		t.Fatalf("shuffle sonrası şarkı sayısı değişmemeli: got=%d want=%d", len(after), len(beforeIDs))
	}

	seen := make(map[string]bool, len(after))
	for _, s := range after {
		seen[s.ID] = true
	}
	for id := range beforeIDs {
		if !seen[id] {
			t.Fatalf("shuffle sonrası %s ID'li şarkı kayboldu: queue=%v", id, extractIDs(after))
		}
	}
	if len(seen) != len(beforeIDs) {
		t.Fatalf("shuffle sonrası beklenmeyen ekstra/mükerrer şarkı: queue=%v", extractIDs(after))
	}
}

// TestShuffle_KeepsCurrentSongIdentity: shuffle sonrası "şu an çalan
// şarkının" ID bazlı kimliğinin korunduğunu doğrular (index değil ID
// bazlı takip tasarım kararının doğruluğunu kanıtlar).
func TestShuffle_KeepsCurrentSongIdentity(t *testing.T) {
	p := buildTestPlayer(t)

	_ = p.Next() // index 0 (ID=1) -> index 1 (ID=2)
	before, _ := p.CurrentSong()
	if before.ID != "2" {
		t.Fatalf("ön koşul hatası: current ID '2' olmalıydı, bulunan: %s", before.ID)
	}

	p.Shuffle()

	after, _ := p.CurrentSong()
	if after.ID != "2" {
		t.Fatalf("shuffle sonrası current şarkı kimliği korunmalıydı (ID=2), bulunan: %s", after.ID)
	}
}

// TestRestoreOrder_ReturnsToOriginal: shuffle sonrası RestoreOrder()'ın
// orijinal (ekleme) sırasına geri döndüğünü doğrular.
func TestRestoreOrder_ReturnsToOriginal(t *testing.T) {
	p := buildTestPlayer(t)
	p.Shuffle()
	p.RestoreOrder()

	// Orijinal sırayı doğrulamak için baştan sona Next() ile geziyoruz.
	first, _ := p.CurrentSong()
	if first.ID != "1" {
		t.Fatalf("RestoreOrder sonrası ilk şarkı '1' olmalı, bulunan: %s", first.ID)
	}
	_ = p.Next()
	second, _ := p.CurrentSong()
	if second.ID != "2" {
		t.Fatalf("RestoreOrder sonrası ikinci şarkı '2' olmalı, bulunan: %s", second.ID)
	}
}
