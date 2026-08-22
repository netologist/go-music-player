package musicplayer

import (
	"fmt"
	"sync"
	"testing"
)

// TestConcurrentAddAndRead: birden fazla goroutine aynı anda playlist'e
// şarkı eklerken, başka goroutine'ler Songs()/Len() ile okuma yaparken
// veri yarışı (race condition) OLMADIĞINI doğrular.
//
// Bu testi `go test -race` ile çalıştırmak KRİTİK: normal çalıştırmada
// race condition'lar sessizce geçebilir (çoğu zaman kod "çalışıyor gibi
// görünür" ama arka planda bozuk state üretir); -race flag'i Go
// runtime'ın belleğe eşzamanlı erişimleri izlemesini sağlar ve gerçek
// bir veri yarışı varsa testi FAIL ettirir.
func TestConcurrentAddAndRead(t *testing.T) {
	p := NewPlaylist("concurrent")

	var wg sync.WaitGroup
	numWriters := 20
	numReaders := 20

	wg.Add(numWriters + numReaders)

	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("song-%d", idx)
			_ = p.AddSong(mustSongNoT(id, fmt.Sprintf("Title %d", idx)))
		}(i)
	}

	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			_ = p.Songs()
			_ = p.Len()
		}()
	}

	wg.Wait()

	if p.Len() != numWriters {
		t.Fatalf("beklenen %d şarkı, bulunan: %d (goroutine'ler arası veri kaybı olabilir)", numWriters, p.Len())
	}
}

// TestConcurrentPlayerControls: aynı anda birden fazla goroutine Next(),
// Shuffle() ve CurrentSong() çağırırken player'ın panik atmadan, tutarlı
// bir state ile ayakta kalmasını doğrular. Sonucun DETERMİNİSTİK olması
// beklenmez (hangi goroutine'in önce çalıştığı garanti değil) — buradaki
// asıl amaç race detector'ın hiçbir veri yarışı raporlamamasıdır.
func TestConcurrentPlayerControls(t *testing.T) {
	pl := NewPlaylist("test")
	for i := 0; i < 10; i++ {
		_ = pl.AddSong(mustSongNoT(fmt.Sprintf("%d", i), fmt.Sprintf("Song %d", i)))
	}
	p := NewPlayer(pl)
	_ = p.Play()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = p.Next()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			p.Shuffle()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = p.CurrentSong()
			_ = p.State()
		}
	}()

	wg.Wait()

	// Test sonunda hâlâ tutarlı (panik atmamış, geçerli bir index'te) bir
	// current song olduğunu doğrula.
	if _, err := p.CurrentSong(); err != nil {
		t.Fatalf("eşzamanlı işlemler sonrası CurrentSong() hata döndü: %v", err)
	}
}

// mustSongNoT: concurrency testlerinde t.Helper() içeren mustSong'u
// goroutine içinden çağırmak güvenli değil (testing.T goroutine-safe
// olsa da *t.Fatalf çağrısı ana goroutine dışında beklenmedik davranabilir);
// bu yüzden hatasız, basit bir yardımcı kullanılır.
func mustSongNoT(id, title string) Song {
	s, _ := NewSong(id, title, "Artist", 0)
	return s
}
