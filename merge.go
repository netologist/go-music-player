package musicplayer

// DuplicatePolicy: iki playlist birleştirilirken aynı ID'ye sahip şarkılar
// için ne yapılacağını belirleyen strateji tipidir.
//
// Neden interface/enum ve neden if-else zinciri değil: yeni bir policy
// eklemek istendiğinde (örn. "keep-longest-duration") mevcut merge
// fonksiyonuna dokunmadan yeni bir const + switch case eklemek yeterli.
// Bu, Open/Closed prensibine uyar ve mülakatta "extensibility'i nasıl
// sağladın?" sorusuna doğrudan cevap verir.
type DuplicatePolicy int

const (
	// KeepFirst: iki playlist'te de varsa, p1'deki (ilk kaynak) versiyon kalır.
	KeepFirst DuplicatePolicy = iota
	// KeepLast: p2'deki (ikinci/son kaynak) versiyon kalır.
	KeepLast
	// KeepBoth: her iki kopya da tutulur (dedup uygulanmaz, iki kez görünür).
	KeepBoth
)

// MergePlaylists: p1 ve p2'yi, HER İKİ kaynaktaki göreceli sırayı koruyarak
// birleştirir ve yeni bir *Playlist döner (p1/p2'yi mutasyona uğratmaz —
// çağıran kod orijinal playlist'leri hâlâ kullanabilsin diye).
//
// "Göreceli sırayı koruma" ne demek: p1 = [A, B, C], p2 = [D, B, E] ise
// sonuç, p1'in kendi içindeki A→B→C sırasını VE p2'nin kendi içindeki
// D→B→E sırasını bozmadan, p1'i baştan, p2'yi p1'in ardından ekleyerek
// oluşturulur. Duplicate (B) policy'e göre bir kez ya da iki kez yer alır.
//
// Zaman karmaşıklığı: O(n + m) — p1 ve p2'nin boyutları toplamı kadar,
// çünkü her şarkı sabit sayıda işlem (map lookup + belki bir append) görür.
// Alan karmaşıklığı: O(n + m) — sonuç playlist + geçici "görülen ID" map'i.
func MergePlaylists(p1, p2 *Playlist, policy DuplicatePolicy) *Playlist {
	// Neden dedup'ı burada KAPALI oluşturuyoruz: duplicate handling'i zaten
	// policy mantığıyla kendimiz (seen/inP2 map'leriyle) explicit olarak
	// yönetiyoruz. Eğer merged.DedupEnabled=true olsaydı ve KeepBoth
	// policy'si çalışsaydı, AddSong kendi dedup kontrolüyle KeepBoth'un
	// "her iki kopyayı da tut" davranışını sessizce bozardı. Playlist'in
	// kendi dedup toggle'ı, kullanıcı sonradan bu merged playlist'e yeni
	// şarkı eklerken kullanılsın diye ayrıca saklanır (aşağıda set edilir).
	merged := NewPlaylist(p1.Name + "+" + p2.Name)

	s1 := p1.Songs()
	s2 := p2.Songs()

	switch policy {
	case KeepBoth:
		// Basit durum: her iki listeyi sırayla ekle, hiçbir şeyi ele.
		for _, s := range s1 {
			_ = merged.AddSong(s) // KeepBoth'ta dedup zaten kapalı davranmalı
		}
		for _, s := range s2 {
			_ = merged.AddSong(s)
		}

	case KeepFirst:
		// p1'i olduğu gibi ekle (kaynak önceliği p1'de).
		seen := make(map[string]bool, len(s1))
		for _, s := range s1 {
			_ = merged.AddSong(s)
			seen[s.ID] = true
		}
		// p2'den sadece p1'de OLMAYAN şarkıları ekle.
		for _, s := range s2 {
			if !seen[s.ID] {
				_ = merged.AddSong(s)
				seen[s.ID] = true
			}
		}

	case KeepLast:
		// Önce p1'i ekle ama p2'de de varsa p1 versiyonunu SONRADAN
		// p2 versiyonuyla değiştireceğiz; en basit ve okunabilir yol:
		// p1'de olup p2'de de olan ID'leri önceden tespit edip p1'den
		// o şarkıları atlamak, sonra p2'yi tam olarak eklemek.
		inP2 := make(map[string]bool, len(s2))
		for _, s := range s2 {
			inP2[s.ID] = true
		}
		for _, s := range s1 {
			if !inP2[s.ID] {
				_ = merged.AddSong(s)
			}
		}
		for _, s := range s2 {
			_ = merged.AddSong(s)
		}
	}

	// Merge işlemi bittikten SONRA, playlist'in gelecekteki AddSong
	// çağrıları için mantıklı bir dedup varsayılanı ayarlıyoruz.
	merged.DedupEnabled = p1.DedupEnabled || p2.DedupEnabled
	return merged
}
