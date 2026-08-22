package musicplayer

import "sync"

// Playlist, sıralı bir şarkı koleksiyonudur.
//
// Veri yapısı kararı (linked list vs slice):
//   - Slice seçildi çünkü gerçek dünyada bir playlist onlarca-binlerce şarkı
//     içerir (milyonlarca değil), bu ölçekte slice'ın cache-locality avantajı
//     ve Go'nun idiomatik koleksiyon tipi olması, linked list'in O(1) orta-nokta
//     silme avantajından daha değerlidir.
//   - Remove/Move işlemleri slice'ta O(n) (shift gerekir); linked list'te bu O(1)
//     olurdu AMA node referansına zaten sahip olmanız gerekir (aksi halde
//     düğümü bulmak yine O(n)). Bizim node referansımız yok, sadece song ID
//     ile arama yapıyoruz, o yüzden linked list'in teorik avantajı pratikte
//     kaybolur.
//   - indexByID map'i O(1) "bu ID playlist'te var mı" / "hangi index'te"
//     sorgularını sağlar; bu, dedup kontrolü ve MoveSong(id, ...) için kritik.
//     Bedeli: her Remove/Move sonrası map'i güncel tutmak (aşağıdaki
//     reindexFrom fonksiyonu bunu yapar).
type Playlist struct {
	mu           sync.RWMutex
	Name         string
	songs        []Song
	indexByID    map[string]int // songID -> songs slice'taki güncel index
	DedupEnabled bool
}

// NewPlaylist bir playlist oluşturur. Functional Options Pattern kullanılarak
// WithDedup veya WithInitialSongs gibi opsiyonlarla esnek şekilde yapılandırılabilir.
func NewPlaylist(name string, opts ...PlaylistOption) *Playlist {
	p := &Playlist{
		Name:         name,
		songs:        make([]Song, 0),
		indexByID:    make(map[string]int),
		DedupEnabled: false,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// AddSong: O(1) amortized (append) + O(1) map insert.
// Dedup açıksa ve ID zaten varsa ErrDuplicateSong döner.
func (p *Playlist) AddSong(s Song) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.DedupEnabled {
		if _, exists := p.indexByID[s.ID]; exists {
			return ErrDuplicateSong
		}
	}

	p.songs = append(p.songs, s)
	p.indexByID[s.ID] = len(p.songs) - 1
	return nil
}

// RemoveSong: ID ile silme. O(n) çünkü silinen index'ten sonraki
// tüm elemanların hem slice'ta kayması hem de map'te index'lerinin
// güncellenmesi gerekir.
func (p *Playlist) RemoveSong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx, exists := p.indexByID[id]
	if !exists {
		return ErrSongNotFound
	}
	return p.removeAtLocked(idx)
}

// RemoveAt: index ile silme (lock alır, internal removeAtLocked'ı çağırır).
func (p *Playlist) RemoveAt(index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < 0 || index >= len(p.songs) {
		return ErrInvalidIndex
	}
	return p.removeAtLocked(index)
}

// removeAtLocked: mutex'in ZATEN alınmış olduğunu varsayar (private helper).
// Neden ayrı bir "Locked" fonksiyon: RemoveSong ve RemoveAt aynı mantığı
// paylaşıyor ama farklı girdilerle (id vs index) çağrılıyor; kilidi iki kez
// almamak (deadlock riski) için ortak mantığı kilitsiz bir helper'a çıkardık.
func (p *Playlist) removeAtLocked(index int) error {
	removedID := p.songs[index].ID
	p.songs = append(p.songs[:index], p.songs[index+1:]...)
	delete(p.indexByID, removedID)
	p.reindexFrom(index)
	return nil
}

// reindexFrom: bir silme/taşıma sonrası index >= from olan tüm şarkıların
// map'teki index bilgisini günceller. O(n) maliyeti, dedup lookup'ının
// O(1) kalabilmesinin bedelidir.
func (p *Playlist) reindexFrom(from int) {
	for i := from; i < len(p.songs); i++ {
		p.indexByID[p.songs[i].ID] = i
	}
}

// MoveSong: bir şarkıyı ID'siyle bulup yeni pozisyona taşır (reorder).
// O(n) — hem eski hem yeni pozisyon arasındaki elemanlar kayar.
func (p *Playlist) MoveSong(id string, newIndex int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldIndex, exists := p.indexByID[id]
	if !exists {
		return ErrSongNotFound
	}
	if newIndex < 0 || newIndex >= len(p.songs) {
		return ErrInvalidPosition
	}
	if oldIndex == newIndex {
		return nil
	}

	song := p.songs[oldIndex]
	// Önce eski pozisyondan çıkar.
	p.songs = append(p.songs[:oldIndex], p.songs[oldIndex+1:]...)
	// Sonra yeni pozisyona ekle (insert).
	p.songs = append(p.songs, Song{})
	copy(p.songs[newIndex+1:], p.songs[newIndex:])
	p.songs[newIndex] = song

	// oldIndex ile newIndex arasındaki her şey kaymış olabilir, en garantili
	// yol tüm map'i baştan kurmak yerine min(old,new)'dan itibaren reindex.
	from := oldIndex
	if newIndex < from {
		from = newIndex
	}
	p.reindexFrom(from)
	return nil
}

// Songs: dışarıya KOPYA döner. Neden kopya: çağıran kod slice'ı elinde
// tutup lock dışında değiştirirse (append, index atama gibi) internal
// state ile senkron bozulur / veri yarışı oluşabilir. Kopya bunu engeller.
func (p *Playlist) Songs() []Song {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]Song, len(p.songs))
	copy(out, p.songs)
	return out
}

// Len: O(1).
func (p *Playlist) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.songs)
}

// Contains: O(1) map lookup.
func (p *Playlist) Contains(id string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.indexByID[id]
	return ok
}

// At: index ile tekil şarkı okuma. O(1).
func (p *Playlist) At(index int) (Song, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if index < 0 || index >= len(p.songs) {
		return Song{}, ErrInvalidIndex
	}
	return p.songs[index], nil
}
