package musicplayer

// MergeStrategy, iki playlist'in şarkı koleksiyonlarını birleştirirken
// çakışmaları ve sıralamayı yöneten Strategy Pattern arayüzüdür.
//
// Neden Strategy Pattern (Open-Closed Principle):
// Switch-case veya if-else dallanması yerine her birleştirme mantığı
// bağımsız bir strateji nesnesi olarak tanımlanır. Yeni bir birleştirme
// stratejisi (örn. en uzun süreliyi seçen, sanatçıya göre önceliklendiren,
// ya da kullanıcı tanımlı özel kurallar) eklemek için mevcut koda dokunulması
// gerekmez.
type MergeStrategy interface {
	Merge(s1, s2 []Song) []Song
}

// MergeStrategyFunc, bağımsız fonksiyonların MergeStrategy arayüzünü sağlamasını
// kolaylaştıran fonksiyonel adapter tipidir.
type MergeStrategyFunc func(s1, s2 []Song) []Song

// Merge, MergeStrategy arayüzünü uygular.
func (f MergeStrategyFunc) Merge(s1, s2 []Song) []Song {
	return f(s1, s2)
}

// keepFirstStrategy: çakışan ID'lerde ilk playlist'teki (s1) şarkıyı korur.
type keepFirstStrategy struct{}

func (keepFirstStrategy) Merge(s1, s2 []Song) []Song {
	seen := make(map[string]bool, len(s1))
	out := make([]Song, 0, len(s1)+len(s2))
	for _, s := range s1 {
		out = append(out, s)
		seen[s.ID] = true
	}
	for _, s := range s2 {
		if !seen[s.ID] {
			out = append(out, s)
			seen[s.ID] = true
		}
	}
	return out
}

// keepLastStrategy: çakışan ID'lerde ikinci playlist'teki (s2) şarkıyı korur.
type keepLastStrategy struct{}

func (keepLastStrategy) Merge(s1, s2 []Song) []Song {
	inS2 := make(map[string]bool, len(s2))
	for _, s := range s2 {
		inS2[s.ID] = true
	}
	out := make([]Song, 0, len(s1)+len(s2))
	for _, s := range s1 {
		if !inS2[s.ID] {
			out = append(out, s)
		}
	}
	out = append(out, s2...)
	return out
}

// keepBothStrategy: hiçbir şarkıyı elemez, tüm kopyaları sırayla ekler.
type keepBothStrategy struct{}

func (keepBothStrategy) Merge(s1, s2 []Song) []Song {
	out := make([]Song, 0, len(s1)+len(s2))
	out = append(out, s1...)
	out = append(out, s2...)
	return out
}

// keepLongestStrategy: çakışan ID'lerde süresi (Duration) daha uzun olan versiyonu seçer.
type keepLongestStrategy struct{}

func (keepLongestStrategy) Merge(s1, s2 []Song) []Song {
	s2Map := make(map[string]Song, len(s2))
	for _, s := range s2 {
		s2Map[s.ID] = s
	}

	seen := make(map[string]bool, len(s1)+len(s2))
	out := make([]Song, 0, len(s1)+len(s2))

	for _, s := range s1 {
		if s2Song, exists := s2Map[s.ID]; exists {
			if s2Song.Duration > s.Duration {
				out = append(out, s2Song)
			} else {
				out = append(out, s)
			}
		} else {
			out = append(out, s)
		}
		seen[s.ID] = true
	}

	for _, s := range s2 {
		if !seen[s.ID] {
			out = append(out, s)
			seen[s.ID] = true
		}
	}
	return out
}

// Standart strateji singleton nesneleri
var (
	// KeepFirst: iki playlist'te de varsa p1'deki (ilk kaynak) versiyon kalır.
	KeepFirst MergeStrategy = keepFirstStrategy{}
	// KeepLast: iki playlist'te de varsa p2'deki (ikinci kaynak) versiyon kalır.
	KeepLast MergeStrategy = keepLastStrategy{}
	// KeepBoth: her iki kopya da tutulur (dedup uygulanmaz, iki kez görünür).
	KeepBoth MergeStrategy = keepBothStrategy{}
	// KeepLongest: çakışan şarkılardan süresi (Duration) daha uzun olan kalır.
	KeepLongest MergeStrategy = keepLongestStrategy{}
)

// MergePlaylists: p1 ve p2'yi verilen MergeStrategy'e göre birleştirir ve
// yeni bir *Playlist döner (p1 ve p2 mutasyona uğramaz).
// strategy nil verilirse varsayılan olarak KeepFirst stratejisi uygulanır.
func MergePlaylists(p1, p2 *Playlist, strategy MergeStrategy) *Playlist {
	if strategy == nil {
		strategy = KeepFirst
	}

	merged := NewPlaylist(p1.Name + "+" + p2.Name)
	s1 := p1.Songs()
	s2 := p2.Songs()

	mergedSongs := strategy.Merge(s1, s2)
	for _, s := range mergedSongs {
		_ = merged.AddSong(s)
	}

	merged.DedupEnabled = p1.DedupEnabled || p2.DedupEnabled
	return merged
}
