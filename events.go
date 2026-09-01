package musicplayer

// EventType describes the kind of event that occurred in the Player.
// Observer Pattern: used to asynchronously/synchronously notify external consumers
// (UI, logger, analytics, audio engine) of internal state changes (state, track, queue, repeat).
type EventType int

const (
	// EventStateChanged is fired when playback state changes (Play, Pause).
	EventStateChanged EventType = iota
	// EventTrackChanged is fired when the current track changes (Next, Previous, Play, etc.).
	EventTrackChanged
	// EventQueueUpdated is fired when the playback queue changes (Shuffle, RestoreOrder, Refresh).
	EventQueueUpdated
	// EventRepeatModeChanged is fired when the repeat mode is updated.
	EventRepeatModeChanged
)

func (e EventType) String() string {
	switch e {
	case EventStateChanged:
		return "StateChanged"
	case EventTrackChanged:
		return "TrackChanged"
	case EventQueueUpdated:
		return "QueueUpdated"
	case EventRepeatModeChanged:
		return "RepeatModeChanged"
	default:
		return "UnknownEvent"
	}
}

// PlayerEvent is the data packet delivered to observers when an event fires.
type PlayerEvent struct {
	Type         EventType
	State        PlaybackState
	CurrentSong  Song
	CurrentIndex int
	RepeatMode   RepeatMode
	QueueLength  int
}

// EventListener is the Observer function type that receives Player events.
type EventListener func(event PlayerEvent)
