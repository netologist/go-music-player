package musicplayer

// MergeStrategy is the Strategy Pattern interface that controls how two playlists'
// song collections are combined, resolving conflicts and determining order.
//
// Why Strategy Pattern (Open-Closed Principle):
// Rather than branching with switch-case or if-else, each merge behaviour is
// encapsulated in its own strategy object. Adding a new strategy (e.g. pick the
// longer version, prioritise by artist, or apply user-defined rules) requires
// zero changes to existing code.
type MergeStrategy interface {
	Merge(s1, s2 []Song) []Song
}

// MergeStrategyFunc is a functional adapter that lets standalone functions satisfy
// the MergeStrategy interface.
type MergeStrategyFunc func(s1, s2 []Song) []Song

// Merge implements the MergeStrategy interface.
func (f MergeStrategyFunc) Merge(s1, s2 []Song) []Song {
	return f(s1, s2)
}

// keepFirstStrategy keeps the s1 (first playlist) version on ID conflicts.
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

// keepLastStrategy keeps the s2 (second playlist) version on ID conflicts.
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

// keepBothStrategy includes every song without deduplication (duplicates appear twice).
type keepBothStrategy struct{}

func (keepBothStrategy) Merge(s1, s2 []Song) []Song {
	out := make([]Song, 0, len(s1)+len(s2))
	out = append(out, s1...)
	out = append(out, s2...)
	return out
}

// keepLongestStrategy picks the version with the longer Duration on ID conflicts.
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

// Standard strategy singletons.
var (
	// KeepFirst: when a song appears in both playlists, the p1 (first source) version wins.
	KeepFirst MergeStrategy = keepFirstStrategy{}
	// KeepLast: when a song appears in both playlists, the p2 (second source) version wins.
	KeepLast MergeStrategy = keepLastStrategy{}
	// KeepBoth: both copies are retained (no dedup; a song may appear twice).
	KeepBoth MergeStrategy = keepBothStrategy{}
	// KeepLongest: on conflict, the version with the longer Duration wins.
	KeepLongest MergeStrategy = keepLongestStrategy{}
)

// MergePlaylists merges p1 and p2 using the provided MergeStrategy and returns
// a new *Playlist (p1 and p2 are not mutated).
// If strategy is nil, KeepFirst is used as the default.
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
