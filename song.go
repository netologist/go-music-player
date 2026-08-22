package musicplayer

import "time"

// Song, sistemdeki en temel domain birimidir.
// ID alanı benzersiz kimlik olarak kullanılır (dedup, lookup, merge policy'lerinde referans noktası).
// Duration, shuffle/istatistik gibi ileri özellikler için ileride kullanılabilir; şimdilik saklanır.
type Song struct {
	ID       string // benzersiz kimlik (örn. UUID ya da katalog ID'si)
	Title    string
	Artist   string
	Duration time.Duration
}

// NewSong, basit bir constructor. Boş ID ile şarkı oluşturmayı engeller
// çünkü ID, dedup ve merge mantığının üzerine kurulduğu tek gerçek anahtar.
func NewSong(id, title, artist string, duration time.Duration) (Song, error) {
	if id == "" {
		return Song{}, ErrEmptySongID
	}
	return Song{
		ID:       id,
		Title:    title,
		Artist:   artist,
		Duration: duration,
	}, nil
}
