package musicplayer

import (
	"errors"
	"math/rand"
	"sync"
)

// PlaybackState: Player'ın o anki oynatma durumu.
type PlaybackState int

const (
	StateStopped PlaybackState = iota
	StatePlaying
	StatePaused
)

// RepeatMode: playlist'in sonuna gelindiğinde (veya Next/Previous
// çağrıldığında) davranışı belirler.
type RepeatMode int

const (
	RepeatOff RepeatMode = iota // sonda dur
	RepeatOne                   // aynı şarkıyı tekrar et
	RepeatAll                   // başa dön
)

var ErrEndOfPlaylist = errors.New("playlist sonuna gelindi (repeat off)")
var ErrNotPlaying = errors.New("player şu an çalmıyor")

// Player, bir Playlist üzerinde oynatma DURUMUNU yönetir.
//
// Tasarım kararı: Player, Playlist'in kendi song slice'ını DEĞİL, kendi
// "queue" kopyasını tutar. Neden: shuffle işlemi playlist'in gerçek
// sırasını bozmamalı (kullanıcı playlist'i başka bir ekranda normal
// sırada görmeye devam edebilmeli). Shuffle sadece Player'ın oynatma
// sırasını etkiler; RestoreOrder ile orijinal playlist sırasına dönülür.
//
// currentSongID (index değil, ID) ile "şu an çalan şarkı" takip edilir.
// Neden index değil ID: shuffle sonrası aynı şarkının index'i değişir;
// eğer sadece index tutsaydık shuffle sonrası yanlış şarkı "current"
// görünürdü. ID bazlı takip, shuffle/reorder sonrası doğru şarkıyı
// bulmamızı garantiler (bkz. syncCurrentIndex).
type Player struct {
	mu            sync.Mutex
	playlist      *Playlist
	queue         []Song
	originalOrder []Song
	currentIndex  int
	currentSongID string
	state         PlaybackState
	repeatMode    RepeatMode
}

// NewPlayer, playlist'in o anki halinden bir snapshot alarak player'ı kurar.
func NewPlayer(playlist *Playlist) *Player {
	songs := playlist.Songs()
	p := &Player{
		playlist:      playlist,
		queue:         songs,
		originalOrder: append([]Song(nil), songs...),
		currentIndex:  0,
		state:         StateStopped,
		repeatMode:    RepeatOff,
	}
	if len(songs) > 0 {
		p.currentSongID = songs[0].ID
	}
	return p
}

// Refresh: playlist dışarıdan değiştiyse (şarkı eklendi/silindi), player'ın
// queue'sunu yeniden senkronize eder. Şu an çalan şarkı hâlâ playlist'te
// varsa "current" konumu korunur; yoksa (silinmişse) başa (index 0) döner.
func (p *Player) Refresh() {
	p.mu.Lock()
	defer p.mu.Unlock()

	songs := p.playlist.Songs()
	p.queue = songs
	p.originalOrder = append([]Song(nil), songs...)
	p.syncCurrentIndexLocked()
}

// syncCurrentIndexLocked: currentSongID'yi queue içinde arayıp currentIndex'i
// buna göre günceller. Bulunamazsa (şarkı silinmiş) index 0'a döner.
// O(n) — playlist büyüklüğüne bağlı, ama sadece shuffle/refresh sonrası
// çağrıldığı için sık çalışan bir yol değil.
func (p *Player) syncCurrentIndexLocked() {
	if len(p.queue) == 0 {
		p.currentIndex = 0
		p.currentSongID = ""
		return
	}
	for i, s := range p.queue {
		if s.ID == p.currentSongID {
			p.currentIndex = i
			return
		}
	}
	p.currentIndex = 0
	p.currentSongID = p.queue[0].ID
}

// Play: durumu Playing yapar. Playlist boşsa hata döner.
func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return ErrEmptyPlaylist
	}
	p.state = StatePlaying
	return nil
}

// Pause: sadece Playing durumundayken anlamlıdır.
func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != StatePlaying {
		return ErrNotPlaying
	}
	p.state = StatePaused
	return nil
}

// Next: repeat moduna göre bir sonraki şarkıya geçer.
//
// Varsayım (console-tabanlı, gerçek zamanlayıcı yok): bu sistemde "şarkı
// bitti" olayı ayrı bir tetikleyici olarak yok; ilerleme sadece Next()
// çağrısıyla simüle ediliyor. Bu yüzden RepeatOne modunda Next() çağrısı
// BİLEREK aynı şarkıda kalır (kullanıcı "tekrarla" dediği için istemsiz
// ilerlemeyi engelliyoruz) — gerçek bir player'da bu, "şarkı bitince aynı
// şarkı yeniden başlar" davranışına karşılık gelir.
func (p *Player) Next() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return ErrEmptyPlaylist
	}

	if p.repeatMode == RepeatOne {
		// Aynı şarkıda kal, sadece "yeniden başlat" anlamına gelir.
		return nil
	}

	if p.currentIndex+1 < len(p.queue) {
		p.currentIndex++
		p.currentSongID = p.queue[p.currentIndex].ID
		return nil
	}

	// Queue'nun sonundayız.
	switch p.repeatMode {
	case RepeatAll:
		p.currentIndex = 0
		p.currentSongID = p.queue[0].ID
		return nil
	default: // RepeatOff
		return ErrEndOfPlaylist
	}
}

// Previous: Next'in simetriği. Başa gelindiğinde RepeatAll ile sona sarar.
func (p *Player) Previous() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return ErrEmptyPlaylist
	}

	if p.repeatMode == RepeatOne {
		return nil
	}

	if p.currentIndex-1 >= 0 {
		p.currentIndex--
		p.currentSongID = p.queue[p.currentIndex].ID
		return nil
	}

	switch p.repeatMode {
	case RepeatAll:
		p.currentIndex = len(p.queue) - 1
		p.currentSongID = p.queue[p.currentIndex].ID
		return nil
	default:
		return ErrEndOfPlaylist
	}
}

// SetRepeatMode: repeat modunu değiştirir.
func (p *Player) SetRepeatMode(mode RepeatMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.repeatMode = mode
}

// Shuffle: Fisher-Yates algoritmasıyla queue'yu karıştırır.
//
// Neden Fisher-Yates: O(n) zaman, O(1) ekstra alan (in-place), ve
// KANITLANMIŞ olarak uniform (her permütasyon eşit olasılıklı) bir
// karıştırma sağlar — naif "rastgele iki elemanı N kere swapla" yaklaşımı
// bias'lı sonuçlar üretebilir.
//
// currentSongID zaten ID bazlı tutulduğu için, shuffle sonrası
// syncCurrentIndexLocked çağrısı "şu an çalan şarkı" kimliğinin
// bozulmamasını garantiler.
func (p *Player) Shuffle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := len(p.queue) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		p.queue[i], p.queue[j] = p.queue[j], p.queue[i]
	}
	p.syncCurrentIndexLocked()
}

// RestoreOrder: shuffle öncesi (orijinal playlist) sırasına döner.
func (p *Player) RestoreOrder() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append([]Song(nil), p.originalOrder...)
	p.syncCurrentIndexLocked()
}

// CurrentSong: o an "çalıyor" olarak işaretli şarkıyı döner.
func (p *Player) CurrentSong() (Song, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return Song{}, ErrEmptyPlaylist
	}
	return p.queue[p.currentIndex], nil
}

// State: o anki playback durumunu döner.
func (p *Player) State() PlaybackState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// RepeatModeValue: o anki repeat modunu döner (test/inceleme kolaylığı için).
func (p *Player) RepeatModeValue() RepeatMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.repeatMode
}

// QueueLen: o anki queue uzunluğunu döner.
func (p *Player) QueueLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.queue)
}

// QueueSongs: o anki oynatma sırasının KOPYASINI döner (inceleme/UI/test
// amaçlı). Songs() metodundaki "neden kopya" gerekçesi burada da geçerli:
// çağıran kod bu slice'ı mutasyona uğratırsa player'ın internal state'i
// bozulmamalı.
func (p *Player) QueueSongs() []Song {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Song, len(p.queue))
	copy(out, p.queue)
	return out
}
