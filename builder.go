package musicplayer

import (
	"errors"
	"time"
)

// SongBuilder, Song domain nesnelerini akıcı (fluent) bir arayüzle
// adım adım inşa etmek için kullanılan Builder Pattern yapısıdır.
//
// Neden Builder Pattern:
// Çok sayıda parametreli nesnelerin oluşturulmasını kolaylaştırır,
// varsayılan değerler (default values) sağlar ve test fixture'larının
// çok daha okunabilir ve esnek yazılabilmesine olanak tanır.
type SongBuilder struct {
	id       string
	title    string
	artist   string
	duration time.Duration
	err      error
}

// NewSongBuilder, verilen ID ile yeni bir SongBuilder başlatır.
func NewSongBuilder(id string) *SongBuilder {
	b := &SongBuilder{
		id:       id,
		duration: 3 * time.Minute, // Mantıklı varsayılan süre
		artist:   "Bilinmeyen Sanatçı",
		title:    "İsimsiz Şarkı",
	}
	if id == "" {
		b.err = ErrEmptySongID
	}
	return b
}

// Title, şarkının başlığını ayarlar.
func (b *SongBuilder) Title(title string) *SongBuilder {
	b.title = title
	return b
}

// Artist, şarkının sanatçısını ayarlar.
func (b *SongBuilder) Artist(artist string) *SongBuilder {
	b.artist = artist
	return b
}

// Duration, şarkının süresini ayarlar.
func (b *SongBuilder) Duration(d time.Duration) *SongBuilder {
	b.duration = d
	return b
}

// Build, yapılandırılmış Song nesnesini doğrular ve üretir.
func (b *SongBuilder) Build() (Song, error) {
	if b.err != nil {
		return Song{}, b.err
	}
	return NewSong(b.id, b.title, b.artist, b.duration)
}

// MustBuild, hata durumunda panic üretir (özellikle test fixture'ları ve demo için).
func (b *SongBuilder) MustBuild() Song {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}
	return s
}

// PlaylistBuilder, Playlist nesnelerini ve başlangıç şarkılarını akıcı (fluent)
// bir arayüzle inşa eden Builder Pattern yapısıdır.
type PlaylistBuilder struct {
	name         string
	dedupEnabled bool
	songs        []Song
	errors       []error
}

// NewPlaylistBuilder, verilen isimle bir PlaylistBuilder başlatır.
func NewPlaylistBuilder(name string) *PlaylistBuilder {
	return &PlaylistBuilder{
		name: name,
	}
}

// WithDedup, playlist için tekrarlanan şarkı kontrolünü yapılandırır.
func (b *PlaylistBuilder) WithDedup(enabled bool) *PlaylistBuilder {
	b.dedupEnabled = enabled
	return b
}

// AddSong, hazır bir Song nesnesini listeye ekler.
func (b *PlaylistBuilder) AddSong(s Song) *PlaylistBuilder {
	b.songs = append(b.songs, s)
	return b
}

// AddNewSong, doğrudan şarkı bilgilerini alarak yeni bir Song oluşturup ekler.
func (b *PlaylistBuilder) AddNewSong(id, title, artist string, duration time.Duration) *PlaylistBuilder {
	s, err := NewSong(id, title, artist, duration)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.songs = append(b.songs, s)
	return b
}

// Build, yapılandırılmış Playlist nesnesini üretir ve doğrulama hatalarını kontrol eder.
func (b *PlaylistBuilder) Build() (*Playlist, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}
	p := NewPlaylist(b.name, WithDedup(b.dedupEnabled))
	for _, s := range b.songs {
		if err := p.AddSong(s); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// MustBuild, hata durumunda panic üretir.
func (b *PlaylistBuilder) MustBuild() *Playlist {
	p, err := b.Build()
	if err != nil {
		panic(err)
	}
	return p
}
