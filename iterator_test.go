package musicplayer

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
)

// TestGenericIterators_SliceSeq: SliceSeq ve SliceSeq2 fonksiyonlarının temel davranışlarını doğrular.
func TestGenericIterators_SliceSeq(t *testing.T) {
	nums := []int{10, 20, 30}

	// SliceSeq2
	var indices []int
	var values []int
	for i, v := range SliceSeq2(nums) {
		indices = append(indices, i)
		values = append(values, v)
	}
	if !slices.Equal(indices, []int{0, 1, 2}) || !slices.Equal(values, nums) {
		t.Fatalf("SliceSeq2 yanlış sonuç: indices=%v, values=%v", indices, values)
	}

	// SliceSeq
	var valuesOnly []int
	for v := range SliceSeq(nums) {
		valuesOnly = append(valuesOnly, v)
	}
	if !slices.Equal(valuesOnly, nums) {
		t.Fatalf("SliceSeq yanlış sonuç: %v", valuesOnly)
	}
}

// TestGenericIterators_LockedSeq_ReleasesLockOnEarlyExit: LockedSeq2'nin break/erken çıkışta
// kilidi kesinlikle serbest bıraktığını (deadlock olmadığını) doğrular.
func TestGenericIterators_LockedSeq_ReleasesLockOnEarlyExit(t *testing.T) {
	var mu sync.Mutex
	items := []string{"a", "b", "c"}

	count := 0
	for i, item := range LockedSeq2(&mu, func() []string { return items }) {
		count++
		if i == 0 && item == "a" {
			break // Erken çıkış
		}
	}

	if count != 1 {
		t.Fatalf("Erken çıkışta count 1 olmalıydı, bulunan: %d", count)
	}

	// Kilit serbest bırakılmış olmalı; Lock alabiliyor muyuz kontrol et
	locked := mu.TryLock()
	if !locked {
		t.Fatalf("LockedSeq2 erken çıkış sonrası kilidi serbest bırakmadı (deadlock riski)")
	}
	mu.Unlock()
}

// TestGenericIterators_FilterAndCollect: Filter, FilterSeq ve Collect fonksiyonlarını doğrular.
func TestGenericIterators_FilterAndCollect(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6}
	isEven := func(n int) bool { return n%2 == 0 }

	evens := Filter(SliceSeq(nums), isEven)
	if !slices.Equal(evens, []int{2, 4, 6}) {
		t.Fatalf("Filter yanlış sonuç: %v", evens)
	}

	evensSeq := FilterSeq(SliceSeq(nums), isEven)
	collected := Collect(evensSeq)
	if !slices.Equal(collected, []int{2, 4, 6}) {
		t.Fatalf("FilterSeq + Collect yanlış sonuç: %v", collected)
	}
}

// TestGenericIterators_Map: Map fonksiyonunun elemanları dönüştürdüğünü doğrular.
func TestGenericIterators_Map(t *testing.T) {
	nums := []int{1, 2, 3}
	strSeq := Map(SliceSeq(nums), func(n int) string {
		return fmt.Sprintf("#%d", n)
	})
	collected := Collect(strSeq)
	if !slices.Equal(collected, []string{"#1", "#2", "#3"}) {
		t.Fatalf("Map sonucu yanlış: %v", collected)
	}
}

// TestGenericIterators_Find: Find fonksiyonunun aranan ilk elemanı bulduğunu doğrular.
func TestGenericIterators_Find(t *testing.T) {
	nums := []int{1, 3, 4, 7, 8}
	found, ok := Find(SliceSeq(nums), func(n int) bool {
		return n%2 == 0
	})
	if !ok || found != 4 {
		t.Fatalf("Find ilk çift sayıyı (4) bulmalıydı: found=%d, ok=%v", found, ok)
	}

	_, ok = Find(SliceSeq(nums), func(n int) bool {
		return n > 100
	})
	if ok {
		t.Fatalf("Find olmayan eleman için false dönmeliydi")
	}
}

// TestPlaylist_AllAndValues: Playlist.All() ve Playlist.Values() iteratörlerini doğrular.
func TestPlaylist_AllAndValues(t *testing.T) {
	p := NewPlaylist("Rock")
	s1 := mustSong(t, "1", "Song A")
	s2 := mustSong(t, "2", "Song B")
	_ = p.AddSong(s1)
	_ = p.AddSong(s2)

	// All() (iter.Seq2)
	var allIDs []string
	var allIndices []int
	for idx, s := range p.All() {
		allIndices = append(allIndices, idx)
		allIDs = append(allIDs, s.ID)
	}
	if !slices.Equal(allIndices, []int{0, 1}) || !slices.Equal(allIDs, []string{"1", "2"}) {
		t.Fatalf("Playlist.All() yanlış sonuç: indices=%v, ids=%v", allIndices, allIDs)
	}

	// Values() (iter.Seq)
	var valIDs []string
	for s := range p.Values() {
		valIDs = append(valIDs, s.ID)
	}
	if !slices.Equal(valIDs, []string{"1", "2"}) {
		t.Fatalf("Playlist.Values() yanlış sonuç: %v", valIDs)
	}
}

// TestPlaylist_ForEach_EarlyExit: ForEach metodunun erken çıkışı desteklediğini doğrular.
func TestPlaylist_ForEach_EarlyExit(t *testing.T) {
	p := NewPlaylist("Rock")
	_ = p.AddSong(mustSong(t, "1", "Song A"))
	_ = p.AddSong(mustSong(t, "2", "Song B"))
	_ = p.AddSong(mustSong(t, "3", "Song C"))

	count := 0
	p.ForEach(func(idx int, song Song) bool {
		count++
		return idx < 1 // 2. elemandan sonra dur
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

// TestPlayer_AllAndValues: Player.All() ve Player.Values() iteratörlerini doğrular.
func TestPlayer_AllAndValues(t *testing.T) {
	pl := NewPlaylist("Mix")
	_ = pl.AddSong(mustSong(t, "1", "A"))
	_ = pl.AddSong(mustSong(t, "2", "B"))

	player := NewPlayer(pl)

	var ids []string
	for _, s := range player.All() {
		ids = append(ids, s.ID)
	}
	if !slices.Equal(ids, []string{"1", "2"}) {
		t.Fatalf("Player.All() yanlış sonuç: %v", ids)
	}

	var valIDs []string
	for s := range player.Values() {
		valIDs = append(valIDs, s.ID)
	}
	if !slices.Equal(valIDs, []string{"1", "2"}) {
		t.Fatalf("Player.Values() yanlış sonuç: %v", valIDs)
	}
}

// TestPlayer_ForEachAndFilter: Player.ForEach ve Player.Filter metodlarını doğrular.
func TestPlayer_ForEachAndFilter(t *testing.T) {
	pl := NewPlaylist("Mix")
	s1, _ := NewSong("1", "A", "Artist1", 3*time.Minute)
	s2, _ := NewSong("2", "B", "Artist2", 4*time.Minute)
	_ = pl.AddSong(s1)
	_ = pl.AddSong(s2)

	player := NewPlayer(pl)

	count := 0
	player.ForEach(func(idx int, s Song) bool {
		count++
		return false // hemen dur
	})
	if count != 1 {
		t.Fatalf("Player.ForEach erken çıkışta 1 kez çalışmalıydı, çalışan: %d", count)
	}

	filtered := player.Filter(func(s Song) bool {
		return s.Artist == "Artist1"
	})
	if len(filtered) != 1 || filtered[0].ID != "1" {
		t.Fatalf("Player.Filter yanlış sonuç: %+v", filtered)
	}
}
