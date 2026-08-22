package musicplayer

import (
	"errors"
	"testing"
	"time"
)

// TestSongBuilder_FluentAndDefaults: SongBuilder'ın varsayılan değerleri ve fluent API'sini doğrular.
func TestSongBuilder_FluentAndDefaults(t *testing.T) {
	s, err := NewSongBuilder("s1").
		Title("Imagine").
		Artist("John Lennon").
		Duration(3 * time.Minute).
		Build()

	if err != nil {
		t.Fatalf("SongBuilder.Build hata verdi: %v", err)
	}
	if s.ID != "s1" || s.Title != "Imagine" || s.Artist != "John Lennon" || s.Duration != 3*time.Minute {
		t.Fatalf("Song alanları beklendiği gibi oluşmadı: %+v", s)
	}
}

// TestSongBuilder_EmptyID_Fails: Boş ID ile SongBuilder hata dönmeli.
func TestSongBuilder_EmptyID_Fails(t *testing.T) {
	_, err := NewSongBuilder("").Title("No ID").Build()
	if !errors.Is(err, ErrEmptySongID) {
		t.Fatalf("ErrEmptySongID bekleniyordu, alınan: %v", err)
	}
}

// TestSongBuilder_MustBuild_PanicsOnInvalid: MustBuild geçersiz durumda panic üretmeli.
func TestSongBuilder_MustBuild_PanicsOnInvalid(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("MustBuild boş ID ile panic üretmeliydi")
		}
	}()
	_ = NewSongBuilder("").MustBuild()
}

// TestPlaylistBuilder_FluentConstruction: PlaylistBuilder ile komple bir playlist inşasını doğrular.
func TestPlaylistBuilder_FluentConstruction(t *testing.T) {
	s1 := NewSongBuilder("1").Title("Song 1").Artist("Artist 1").MustBuild()

	p, err := NewPlaylistBuilder("Rock Anthems").
		WithDedup(true).
		AddSong(s1).
		AddNewSong("2", "Song 2", "Artist 2", 4*time.Minute).
		Build()

	if err != nil {
		t.Fatalf("PlaylistBuilder.Build hata verdi: %v", err)
	}

	if p.Name != "Rock Anthems" || !p.DedupEnabled || p.Len() != 2 {
		t.Fatalf("Playlist beklenen özelliklerde oluşmadı: Name=%s, Dedup=%v, Len=%d", p.Name, p.DedupEnabled, p.Len())
	}
}

// TestPlaylistBuilder_DuplicateSongError: Dedup açıkken mükerrer şarkı eklenirse Build hata dönmeli.
func TestPlaylistBuilder_DuplicateSongError(t *testing.T) {
	s1 := NewSongBuilder("1").Title("Song 1").MustBuild()

	_, err := NewPlaylistBuilder("Test").
		WithDedup(true).
		AddSong(s1).
		AddSong(s1).
		Build()

	if !errors.Is(err, ErrDuplicateSong) {
		t.Fatalf("ErrDuplicateSong bekleniyordu, alınan: %v", err)
	}
}

// TestPlaylistBuilder_InvalidSongErrorAccumulation: Hatalı şarkılar Build sırasında hata biriktirmeli.
func TestPlaylistBuilder_InvalidSongErrorAccumulation(t *testing.T) {
	_, err := NewPlaylistBuilder("Test").
		AddNewSong("", "No ID", "Artist", 1*time.Minute).
		Build()

	if !errors.Is(err, ErrEmptySongID) {
		t.Fatalf("ErrEmptySongID bekleniyordu, alınan: %v", err)
	}
}
