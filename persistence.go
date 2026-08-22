package musicplayer

import (
	"encoding/json"
	"os"
)

// playlistSnapshot: Playlist struct'ının serialize edilebilir "düz" hali.
//
// Neden Playlist'i doğrudan json.Marshal etmiyoruz: Playlist'in internal
// state'i (mu sync.RWMutex, indexByID map'i) serialize edilmemeli — mutex
// zaten serialize edilemez (kilitlenebilir), indexByID ise songs slice'ından
// TÜRETİLEBİLEN (derived) bir veri, yani onu diske yazmak gereksiz veri
// tekrarı olur ve yükleme sırasında songs ile senkron kalmama riski taşır.
// Bu yüzden sadece "gerçek kaynak" olan Name/Songs/DedupEnabled kaydedilir,
// indexByID yükleme sırasında AddSong çağrılarıyla yeniden inşa edilir.
type playlistSnapshot struct {
	Name         string `json:"name"`
	Songs        []Song `json:"songs"`
	DedupEnabled bool   `json:"dedup_enabled"`
}

// SaveToFile: playlist'i JSON olarak diske yazar.
//
// Neden JSON (ve neden encoding/json): format insan-okunabilir, standart
// kütüphane yeterli (üçüncü parti bağımlılık gerekmiyor), ve Song struct'ı
// zaten basit/düz alanlardan oluştuğu için karmaşık bir serializer'a
// ihtiyaç yok. Daha yüksek performans gerekseydi (binlerce playlist,
// sık kayıt) gob ya da protobuf düşünülebilirdi, ama bu ölçekte JSON'un
// okunabilirlik avantajı performans farkından daha değerli.
//
// Concurrency notu: RLock alınır çünkü sadece okuma yapıyoruz; bu sayede
// kaydetme sırasında başka goroutine'ler playlist'i OKUMAYA devam edebilir,
// sadece eşzamanlı YAZMA işlemleri (Add/Remove/Move) bloklanır.
func (p *Playlist) SaveToFile(path string) error {
	p.mu.RLock()
	snap := playlistSnapshot{
		Name:         p.Name,
		Songs:        append([]Song(nil), p.songs...),
		DedupEnabled: p.DedupEnabled,
	}
	p.mu.RUnlock()

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadPlaylistFromFile: diskten bir playlist okur ve indexByID map'ini
// AddSong çağrılarıyla YENİDEN İNŞA eder (yukarıdaki yorumda açıklanan
// "derived state'i diske yazma" sorununu böyle çözüyoruz).
func LoadPlaylistFromFile(path string) (*Playlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var snap playlistSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}

	// Dedup'ı yükleme sırasında kapalı tutuyoruz ki, diskte zaten var olan
	// (belki dedup kapalıyken kaydedilmiş) tekrar eden şarkılar sessizce
	// kaybolmasın; asıl DedupEnabled değerini en son ayarlıyoruz.
	pl := NewPlaylist(snap.Name, false)
	for _, s := range snap.Songs {
		if err := pl.AddSong(s); err != nil {
			return nil, err
		}
	}
	pl.DedupEnabled = snap.DedupEnabled
	return pl, nil
}
