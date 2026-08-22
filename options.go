package musicplayer

import "math/rand/v2"

// PlaylistOption, Playlist nesnesini yapılandırmak için kullanılan fonksiyonel opsiyon tipidir.
// Functional Options Pattern: constructor parametrelerinin çoğalmasını ve bool parametrelerin
// ("boolean trap" anti-pattern) yarattığı okunabilirlik sorununu önler.
type PlaylistOption func(*Playlist)

// WithDedup, playlist'in tekrarlanan şarkı kontrolünü aktif veya pasif yapar.
func WithDedup(enabled bool) PlaylistOption {
	return func(p *Playlist) {
		p.DedupEnabled = enabled
	}
}

// WithInitialSongs, playlist oluşturulurken başlangıç şarkılarının eklenmesini sağlar.
func WithInitialSongs(songs ...Song) PlaylistOption {
	return func(p *Playlist) {
		for _, s := range songs {
			_ = p.AddSong(s)
		}
	}
}

// PlayerOption, Player nesnesini yapılandırmak için kullanılan fonksiyonel opsiyon tipidir.
type PlayerOption func(*Player)

// WithRepeatMode, player'ın başlangıç tekrar modunu ayarlar.
func WithRepeatMode(mode RepeatMode) PlayerOption {
	return func(p *Player) {
		p.repeatMode = mode
	}
}

// WithRandSource, shuffle işleminde kullanılacak rastgele sayı üreticisini inject eder.
// Dependency Injection: testlerde deterministik shuffle sağlamak için özel *rand.Rand verilebilir.
func WithRandSource(r *rand.Rand) PlayerOption {
	return func(p *Player) {
		p.rng = r
	}
}

// WithEventListener, player'a bir olay dinleyicisi ekler (Observer Pattern).
func WithEventListener(l EventListener) PlayerOption {
	return func(p *Player) {
		p.Subscribe(l)
	}
}
