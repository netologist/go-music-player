package musicplayer

import "errors"

// Sentinel errors allow callers to distinguish specific failure kinds via errors.Is
// (e.g. "song not found" vs "invalid index" can be handled differently).
var (
	ErrEmptySongID     = errors.New("song id cannot be empty")
	ErrSongNotFound    = errors.New("song not found")
	ErrDuplicateSong   = errors.New("a song with this id already exists in the playlist")
	ErrInvalidIndex    = errors.New("invalid index")
	ErrEmptyPlaylist   = errors.New("playlist is empty")
	ErrInvalidPosition = errors.New("invalid position")
)
