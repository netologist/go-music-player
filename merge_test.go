package musicplayer

import "testing"

func buildTestPlaylists(t *testing.T) (*Playlist, *Playlist) {
	t.Helper()
	p1 := NewPlaylist("P1", false)
	_ = p1.AddSong(mustSong(t, "A", "Song A"))
	_ = p1.AddSong(mustSong(t, "B", "Song B - v1"))
	_ = p1.AddSong(mustSong(t, "C", "Song C"))

	p2 := NewPlaylist("P2", false)
	_ = p2.AddSong(mustSong(t, "D", "Song D"))
	_ = p2.AddSong(mustSong(t, "B", "Song B - v2")) // B, p1'de de var (çakışma)
	_ = p2.AddSong(mustSong(t, "E", "Song E"))

	return p1, p2
}

// TestMerge_KeepFirst: çakışan ID'de (B) p1'deki versiyonun kazandığını
// VE göreceli sıranın korunduğunu doğrular.
func TestMerge_KeepFirst(t *testing.T) {
	p1, p2 := buildTestPlaylists(t)
	merged := MergePlaylists(p1, p2, KeepFirst)

	songs := merged.Songs()
	ids := extractIDs(songs)
	// Beklenen: p1 sırası korunur [A,B,C], sonra p2'den B hariç [D,E] eklenir.
	want := []string{"A", "B", "C", "D", "E"}
	assertIDOrder(t, ids, want)

	// B'nin p1 versiyonu (title) kazanmış olmalı.
	b := findByID(songs, "B")
	if b.Title != "Song B - v1" {
		t.Fatalf("KeepFirst policy'de B'nin p1 versiyonu kalmalıydı, bulunan title: %s", b.Title)
	}
}

// TestMerge_KeepLast: çakışan ID'de p2'deki versiyonun kazandığını
// VE sıranın korunduğunu doğrular.
func TestMerge_KeepLast(t *testing.T) {
	p1, p2 := buildTestPlaylists(t)
	merged := MergePlaylists(p1, p2, KeepLast)

	songs := merged.Songs()
	ids := extractIDs(songs)
	// Beklenen: p1'den B çıkarılmış [A,C], sonra p2 tam olarak [D,B,E] eklenir.
	want := []string{"A", "C", "D", "B", "E"}
	assertIDOrder(t, ids, want)

	b := findByID(songs, "B")
	if b.Title != "Song B - v2" {
		t.Fatalf("KeepLast policy'de B'nin p2 versiyonu kalmalıydı, bulunan title: %s", b.Title)
	}
}

// TestMerge_KeepBoth: çakışan ID'nin İKİ KEZ de yer aldığını doğrular.
func TestMerge_KeepBoth(t *testing.T) {
	p1, p2 := buildTestPlaylists(t)
	merged := MergePlaylists(p1, p2, KeepBoth)

	songs := merged.Songs()
	if len(songs) != 6 {
		t.Fatalf("KeepBoth'ta toplam 6 şarkı beklenir (3+3, hiçbiri elenmez), bulunan: %d", len(songs))
	}

	ids := extractIDs(songs)
	want := []string{"A", "B", "C", "D", "B", "E"}
	assertIDOrder(t, ids, want)
}

// TestMerge_DoesNotMutateOriginals: merge işleminin p1/p2'yi DEĞİŞTİRMEDİĞİNİ
// doğrular — orijinal playlist'ler merge sonrası hâlâ kullanılabilir olmalı.
func TestMerge_DoesNotMutateOriginals(t *testing.T) {
	p1, p2 := buildTestPlaylists(t)
	originalP1Len := p1.Len()
	originalP2Len := p2.Len()

	_ = MergePlaylists(p1, p2, KeepFirst)

	if p1.Len() != originalP1Len || p2.Len() != originalP2Len {
		t.Fatalf("merge, orijinal playlist'lerin uzunluğunu değiştirmiş: p1=%d p2=%d", p1.Len(), p2.Len())
	}
}

// --- test yardımcı fonksiyonları ---

func extractIDs(songs []Song) []string {
	ids := make([]string, len(songs))
	for i, s := range songs {
		ids[i] = s.ID
	}
	return ids
}

func assertIDOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("uzunluk uyuşmuyor: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sıra uyuşmuyor: got=%v want=%v", got, want)
		}
	}
}

func findByID(songs []Song, id string) Song {
	for _, s := range songs {
		if s.ID == id {
			return s
		}
	}
	return Song{}
}
