package main

import (
	"context"
	"fmt"
	"os"
	"time"

	mp "musicplayer"
)

func main() {
	// 1) Build two playlists with the Builder Pattern and add songs.
	rock := mp.NewPlaylistBuilder("Rock Classics").
		WithDedup(true).
		AddNewSong("r1", "Bohemian Rhapsody", "Queen", 6*time.Minute).
		AddNewSong("r2", "Sweet Child O' Mine", "Guns N' Roses", 5*time.Minute).
		AddNewSong("r3", "Back In Black", "AC/DC", 4*time.Minute).
		MustBuild()

	indie := mp.NewPlaylistBuilder("Indie Discoveries").
		WithDedup(true).
		AddNewSong("i1", "Midnight City", "M83", 4*time.Minute).
		AddNewSong("r2", "Sweet Child O' Mine", "Guns N' Roses", 5*time.Minute). // conflicts with r2 above
		AddNewSong("i2", "Feel Good Inc.", "Gorillaz", 3*time.Minute).
		MustBuild()

	fmt.Println("Rock playlist:", ids(rock))
	fmt.Println("Indie playlist:", ids(indie))

	// 2) Merge with the KeepFirst policy (rock's version wins on conflict).
	merged := mp.MergePlaylists(rock, indie, mp.KeepFirst)
	fmt.Println("Merged (KeepFirst):", ids(merged))

	// Iterate with iter.Seq2 (range-over-func, Go 1.23+).
	fmt.Print("Merged songs (iter.Seq2): ")
	for _, s := range merged.All() {
		fmt.Printf("[%s: %s] ", s.ID, s.Title)
	}
	fmt.Println()

	// 3) Create a Player, attach an Observer listener, and start playback.
	player := mp.NewPlayer(merged, mp.WithEventListener(func(e mp.PlayerEvent) {
		fmt.Printf("  [Event: %s] Song: %s, State: %v\n", e.Type, e.CurrentSong.Title, e.State)
	}))
	must(player.Play())
	cur, _ := player.CurrentSong()
	fmt.Println("Now playing:", cur.Title)
	must(player.Next())
	cur, _ = player.CurrentSong()
	fmt.Println("After Next:", cur.Title)

	player.SetRepeatMode(mp.RepeatAll)
	player.Shuffle()
	cur, _ = player.CurrentSong()
	fmt.Println("After Shuffle, now playing:", cur.Title)

	// 4) Persist with the Repository Pattern.
	repo, err := mp.NewJSONFileRepository("/tmp/musicplayer_repo")
	if err != nil {
		fmt.Println("repo error:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := repo.Save(ctx, merged); err != nil {
		fmt.Println("repo save error:", err)
		os.Exit(1)
	}
	fmt.Println("Playlist saved via Repository (JSONFileRepository).")

	// 5) Reload from the repository and verify.
	loaded, err := repo.Load(ctx, merged.Name)
	if err != nil {
		fmt.Println("repo load error:", err)
		os.Exit(1)
	}
	fmt.Println("Playlist loaded from repository:", ids(loaded))
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
