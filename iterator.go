package musicplayer

// SongIterator, şarkı koleksiyonları üzerinde adım adım güvenli ve
// bellek dostu gezinme sağlayan Iterator Pattern arayüzüdür.
//
// Neden Iterator Pattern:
// Koleksiyonun iç veri yapısını (slice) dış dünyaya açmadan, çağıran koda
// tek tek eleman tüketme imkânı sunar. Tüm slice'ı tek seferde kopyalamak (Songs())
// yerine, ihtiyaç duyuldukça eleman okunmasını ve erken çıkış yapılabilmesini sağlar.
type SongIterator interface {
	// HasNext, sırada tüketilmeyi bekleyen şarkı olup olmadığını kontrol eder.
	HasNext() bool
	// Next, sıradaki şarkıyı döner ve imleci bir ileri kaydırır.
	// Kalan şarkı yoksa (Song{}, false) döner.
	Next() (Song, bool)
	// Peek, imleci ilerletmeden sıradaki şarkıyı okur.
	Peek() (Song, bool)
	// Reset, iterator'ı listenin başına sıfırlar.
	Reset()
	// Total, koleksiyondaki toplam şarkı sayısını döner.
	Total() int
	// Remaining, henüz okunmamış kalan şarkı sayısını döner.
	Remaining() int
}

// sliceSongIterator, Song slice'ı üzerinde çalışan somut SongIterator implementasyonudur.
type sliceSongIterator struct {
	items []Song
	index int
}

// NewSongIterator, verilen şarkı dilimi üzerinde bir SongIterator oluşturur.
func NewSongIterator(songs []Song) SongIterator {
	cp := make([]Song, len(songs))
	copy(cp, songs)
	return &sliceSongIterator{
		items: cp,
		index: 0,
	}
}

func (it *sliceSongIterator) HasNext() bool {
	return it.index < len(it.items)
}

func (it *sliceSongIterator) Next() (Song, bool) {
	if !it.HasNext() {
		return Song{}, false
	}
	s := it.items[it.index]
	it.index++
	return s, true
}

func (it *sliceSongIterator) Peek() (Song, bool) {
	if !it.HasNext() {
		return Song{}, false
	}
	return it.items[it.index], true
}

func (it *sliceSongIterator) Reset() {
	it.index = 0
}

func (it *sliceSongIterator) Total() int {
	return len(it.items)
}

func (it *sliceSongIterator) Remaining() int {
	rem := len(it.items) - it.index
	if rem < 0 {
		return 0
	}
	return rem
}
