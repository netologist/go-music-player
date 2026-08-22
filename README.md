# MusicPlayer

Modern, eşzamanlılığa uygun (thread-safe), yüksek test kapsamına sahip ve sektör standardı tasarım desenleri (Design Patterns) ile yazılım mühendisliği pratikleri (Best Practices) uygulanarak geliştirilmiş Go müzik çalar kütüphanesi ve örnek uygulaması.

---

## 📑 İçindekiler
- [Mimari ve Tasarım Desenleri](#-mimari-ve-tasarım-desenleri)
- [Yazılım Mühendisliği Pratikleri](#-yazılım-mühendisliği-pratikleri)
- [Proje Dosya Yapısı](#-proje-dosya-yapısı)
- [Kullanım Örnekleri](#-kullanım-örnekleri)
  - [1. Builder Pattern ile Playlist ve Şarkı Oluşturma](#1-builder-pattern-ile-playlist-ve-şarkı-oluşturma)
  - [2. Functional Options Pattern ile Yapılandırma](#2-functional-options-pattern-ile-yapılandırma)
  - [3. Strategy Pattern ile Playlist Birleştirme (Merge)](#3-strategy-pattern-ile-playlist-birleştirme-merge)
  - [4. Observer Pattern ile Olay Dinleme](#4-observer-pattern-ile-olay-dinleme)
  - [5. Repository ve Adapter Pattern ile Kalıcılık (Persistence)](#5-repository-ve-adapter-pattern-ile-kalıcılık-persistence)
  - [6. Iterator Pattern ile Bellek Dostu Gezinme](#6-iterator-pattern-ile-bellek-dostu-gezinme)
- [Test ve Doğrulama](#-test-ve-doğrulama)

---

## 🏛 Mimari ve Tasarım Desenleri

Projede uygulanan 6 temel tasarım deseni ve çözdükleri mimari problemler:

| Desen | Dosya | Çözülen Problem / Sağlanan Fayda |
| :--- | :--- | :--- |
| **Functional Options Pattern** | `options.go` | Constructor'larda `bool` bayrakların yarattığı okunabilirlik sorununu (`Boolean Trap`) çözer, opsiyonel parametrelerin esnek ve geriye uyumlu yönetilmesini sağlar. |
| **Strategy Pattern** | `merge.go` | Çakışan şarkı birleştirme algoritmalarını `switch-case` dallanmasından kurtarır. `Open-Closed Principle (OCP)` uyarınca çekirdek koda dokunmadan yeni stratejiler eklenmesini sağlar. |
| **Observer Pattern** | `events.go`, `player.go` | `Player` bileşenini UI, loglama, ses motoru ve analitik katmanlarından tamamen soyutlar. Durum değişikliklerini olay güdümlü (`Event-Driven`) iletir. |
| **Repository & Adapter Pattern** | `repository.go` | `Dependency Inversion Principle (DIP)` gereği domain modelini somut dosya I/O işlemlerinden ayırır. `JSONFileRepository` ve `MemoryRepository` adaptörleri sunar. |
| **Iterator Pattern** | `iterator.go`, `playlist.go` | Koleksiyonun iç yapısını dışarı açmadan ve her seferinde tüm listeyi kopyalama (`Songs()`) bellek maliyeti oluşturmadan eleman bazlı akış/gezinme sağlar. |
| **Builder Pattern** | `builder.go` | Çok sayıda alana sahip `Song` ve `Playlist` nesnelerinin akıcı metot zinciri (`fluent interface`) ile inşa edilmesini, doğrulanmasını ve test fixture'larının kolay oluşturulmasını sağlar. |

---

## 🛠 Yazılım Mühendisliği Pratikleri

1. **Eşzamanlılık Güvenliği (Thread-Safety & Race Condition Prevention):**
   - `Playlist` içi okuma/yazma ayrımı `sync.RWMutex` ile korunur.
   - `Player` oynatma durumu `sync.Mutex` ile atomik yönetilir.
   - Go Race Detector (`-race`) ile tüm eşzamanlı okuma/yazma senaryoları test edilmiştir.

2. **Kilitlenmesiz Olay Dağıtımı (Deadlock-Free Notification):**
   - `Player.Subscribe` dinleyicileri çağrılırken, callback içinde `player.State()` gibi metotların çağrılması durumunda oluşabilecek `deadlock` riskini önlemek için dinleyici listesi kilit altında kopyalanır ve mutex **dışında** tetiklenir.

3. **Atomik Dosya Yazma (Atomic Write via Temp File + Rename):**
   - `JSONFileRepository` ve `SaveToFile`, dosyayı doğrudan hedef yola yazmak yerine önce `.tmp` uzantılı geçici bir dosyaya yazar, ardından işletim sistemi seviyesinde atomik `os.Rename` işlemi yapar. Bu sayede yazma anında oluşabilecek elektrik kesintisi veya çökme durumlarında dosya bozulması (`data corruption`) engellenir.

4. **Modern Go Standartları (`math/rand/v2` & Dependency Injection):**
   - Eski global kilitli `math/rand` yerine Go 1.22+ ile gelen `math/rand/v2` kullanılmıştır.
   - `WithRandSource` opsiyonu ile deterministik rastgele sayı kaynağı enjekte edilebilir, böylece `Shuffle` testleri deterministik ve güvenli şekilde doğrulanır.

5. **Erken Çıkışlı & Sıfır Kopyalı Gezinti (Non-Allocating Traversal):**
   - `Playlist.ForEach` fonksiyonu kilit altında çalışır ve `false` dönüldüğünde döngüyü erken sonlandırır (`short-circuiting`). Ekstra slice belleği tahsis etmez ($O(1)$ ek bellek).

---

## 📂 Proje Dosya Yapısı

```text
.
├── builder.go            # Builder Pattern (SongBuilder, PlaylistBuilder)
├── builder_test.go       # Builder birim testleri
├── concurrency_test.go   # Eşzamanlı okuma/yazma ve race detector testleri
├── errors.go             # Sentinel hata tanımları (ErrDuplicateSong, ErrSongNotFound vb.)
├── events.go             # Observer Pattern (EventType, PlayerEvent, EventListener)
├── go.mod                # Go modül tanımı
├── iterator.go           # Iterator Pattern (SongIterator, sliceSongIterator)
├── iterator_test.go      # Iterator, ForEach ve Filter birim testleri
├── merge.go              # Strategy Pattern (MergeStrategy, KeepFirst, KeepLast, KeepBoth, KeepLongest)
├── merge_test.go         # Merge stratejileri birim testleri
├── options.go            # Functional Options Pattern (PlaylistOption, PlayerOption)
├── persistence.go        # Geriye uyumlu JSON snapshot ve dosya helper'ları
├── persistence_test.go   # Dosya ve Repository testleri
├── player.go             # Oynatıcı mantığı, state machine, shuffle ve observer entegrasyonu
├── player_test.go        # Oynatıcı durum geçişleri ve event testleri
├── playlist.go           # Playlist domain modeli, dedup index map ve concurrency
├── playlist_test.go      # Playlist operasyonları testleri
├── song.go               # Song temel domain struct'ı ve constructor
└── cmd/
    └── demo/
        └── main.go       # Tüm desenlerin birlikte çalıştığı uçtan uca demo uygulaması
```

---

## 💻 Kullanım Örnekleri

### 1. Builder Pattern ile Playlist ve Şarkı Oluşturma

```go
package main

import (
	"time"
	mp "musicplayer"
)

func main() {
	// Fluent API ile Playlist ve Şarkı inşası
	playlist := mp.NewPlaylistBuilder("Favori Rock").
		WithDedup(true).
		AddNewSong("r1", "Bohemian Rhapsody", "Queen", 6*time.Minute).
		AddNewSong("r2", "Sweet Child O' Mine", "Guns N' Roses", 5*time.Minute).
		MustBuild()
}
```

### 2. Functional Options Pattern ile Yapılandırma

```go
// Playlist oluştururken boolean trap olmadan açık yapılandırma:
pl := mp.NewPlaylist("Indie Mix", 
	mp.WithDedup(true),
	mp.WithInitialSongs(song1, song2),
)

// Player oluştururken başlangıç modunu ve bağımlılıkları belirleme:
player := mp.NewPlayer(pl, 
	mp.WithRepeatMode(mp.RepeatAll),
)
```

### 3. Strategy Pattern ile Playlist Birleştirme (Merge)

```go
// Hazır stratejiler: KeepFirst, KeepLast, KeepBoth, KeepLongest
mergedFirst := mp.MergePlaylists(rockList, indieList, mp.KeepFirst)
mergedLongest := mp.MergePlaylists(rockList, indieList, mp.KeepLongest)

// İsteğe bağlı özel strateji (Custom Strategy) fonksiyonu:
customStrategy := mp.MergeStrategyFunc(func(s1, s2 []mp.Song) []mp.Song {
	// Özel filtreleme / birleştirme kuralı
	return append(s1, s2...)
})
mergedCustom := mp.MergePlaylists(rockList, indieList, customStrategy)
```

### 4. Observer Pattern ile Olay Dinleme

```go
player := mp.NewPlayer(playlist)

// Durum, parça, kuyruk ve tekrar modu değişikliklerine abone olma:
unsubscribe := player.Subscribe(func(e mp.PlayerEvent) {
	switch e.Type {
	case mp.EventStateChanged:
		println("Oynatma durumu değişti:", e.State)
	case mp.EventTrackChanged:
		println("Şu an çalan şarkı:", e.CurrentSong.Title)
	case mp.EventQueueUpdated:
		println("Kuyruk güncellendi / karıştırıldı")
	}
})
defer unsubscribe()

player.Play()
player.Next()
player.Shuffle()
```

### 5. Repository ve Adapter Pattern ile Kalıcılık (Persistence)

```go
ctx := context.Background()

// 1. JSON Dosya Sistemi Adaptörü (Atomic write destekli)
fileRepo, _ := mp.NewJSONFileRepository("./data/playlists")
_ = fileRepo.Save(ctx, playlist)
loaded, _ := fileRepo.Load(ctx, "Favori Rock")

// 2. In-Memory Adaptörü (Test ve önbellekleme için)
memRepo := mp.NewMemoryRepository()
_ = memRepo.Save(ctx, playlist)
```

### 6. Iterator Pattern ile Bellek Dostu Gezinme

```go
// 1. Iterator ile adım adım tüketim:
it := playlist.Iterator()
for it.HasNext() {
	song, _ := it.Next()
	println(song.Title)
}

// 2. ForEach ile erken çıkışlı (short-circuit) gezinme (0 ek bellek):
playlist.ForEach(func(idx int, song mp.Song) bool {
	if song.Artist == "Queen" {
		println("Bulundu:", song.Title)
		return false // Aramayı durdur
	}
	return true // Devam et
})
```

---

## 🧪 Test ve Doğrulama

Tüm birim testlerini ve eşzamanlılık yarış durumu (race condition) denetleyicisini çalıştırmak için:

```bash
go test -v -race ./...
```

Uçtan uca demo uygulamasını çalıştırmak için:

```bash
go run ./cmd/demo
```
