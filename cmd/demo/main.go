package main

import (
	"context"
	"fmt"
	"os"
	"time"

	mp "musicplayer"
)

func main() {
	// 1) İki playlist oluştur, şarkı ekle.
	rock := mp.NewPlaylist("Rock Klasikleri", mp.WithDedup(true))
	must(rock.AddSong(song("r1", "Bohemian Rhapsody", "Queen")))
	must(rock.AddSong(song("r2", "Sweet Child O' Mine", "Guns N' Roses")))
	must(rock.AddSong(song("r3", "Back In Black", "AC/DC")))

	indie := mp.NewPlaylist("Indie Keşifler", mp.WithDedup(true))
	must(indie.AddSong(song("i1", "Midnight City", "M83")))
	must(indie.AddSong(song("r2", "Sweet Child O' Mine", "Guns N' Roses"))) // r2 ile çakışıyor
	must(indie.AddSong(song("i2", "Feel Good Inc.", "Gorillaz")))

	fmt.Println("Rock playlist:", ids(rock))
	fmt.Println("Indie playlist:", ids(indie))

	// 2) Merge et - KeepFirst policy ile (rock'taki versiyon kazanır).
	merged := mp.MergePlaylists(rock, indie, mp.KeepFirst)
	fmt.Println("Merged (KeepFirst):", ids(merged))

	// 3) Player oluştur, Observer listener bağla, oynatmayı başlat.
	player := mp.NewPlayer(merged, mp.WithEventListener(func(e mp.PlayerEvent) {
		fmt.Printf("  [Event: %s] Şarkı: %s, State: %v\n", e.Type, e.CurrentSong.Title, e.State)
	}))
	must(player.Play())
	cur, _ := player.CurrentSong()
	fmt.Println("Çalıyor:", cur.Title)
	must(player.Next())
	cur, _ = player.CurrentSong()
	fmt.Println("Next sonrası:", cur.Title)

	player.SetRepeatMode(mp.RepeatAll)
	player.Shuffle()
	cur, _ = player.CurrentSong()
	fmt.Println("Shuffle sonrası çalıyor:", cur.Title)

	// 4) Repository Pattern ile kaydet ve yükle.
	repo, err := mp.NewJSONFileRepository("/tmp/musicplayer_repo")
	if err != nil {
		fmt.Println("repo hatası:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := repo.Save(ctx, merged); err != nil {
		fmt.Println("repo kaydetme hatası:", err)
		os.Exit(1)
	}
	fmt.Println("Playlist Repository (JSONFileRepository) ile kaydedildi.")

	// 5) Repository'den geri yükle ve doğrula.
	loaded, err := repo.Load(ctx, merged.Name)
	if err != nil {
		fmt.Println("repo yükleme hatası:", err)
		os.Exit(1)
	}
	fmt.Println("Repository'den yüklenen playlist:", ids(loaded))
}

func song(id, title, artist string) mp.Song {
	s, err := mp.NewSong(id, title, artist, 3*time.Minute)
	if err != nil {
		panic(err)
	}
	return s
}

func ids(p *mp.Playlist) []string {
	songs := p.Songs()
	out := make([]string, len(songs))
	for i, s := range songs {
		out[i] = s.ID
	}
	return out
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
