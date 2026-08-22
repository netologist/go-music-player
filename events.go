package musicplayer

// EventType, oynatıcıda (Player) meydana gelen olayların tipini belirtir.
// Observer Pattern: Player'ın iç durum değişikliklerini (state, track, queue, repeat)
// dış dünyaya (UI, logger, analytics, audio engine) asenkron/senkron bildirmek için kullanılır.
type EventType int

const (
	// EventStateChanged: Oynatma durumu değiştiğinde (Play, Pause) tetiklenir.
	EventStateChanged EventType = iota
	// EventTrackChanged: Oynatılan şarkı değiştiğinde (Next, Previous, Play vb.) tetiklenir.
	EventTrackChanged
	// EventQueueUpdated: Oynatma sırası/koleksiyon değiştiğinde (Shuffle, RestoreOrder, Refresh) tetiklenir.
	EventQueueUpdated
	// EventRepeatModeChanged: Repeat modu güncellendiğinde tetiklenir.
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

// PlayerEvent, olay tetiklendiğinde dinleyicilere (observer) iletilen veri paketidir.
type PlayerEvent struct {
	Type         EventType
	State        PlaybackState
	CurrentSong  Song
	CurrentIndex int
	RepeatMode   RepeatMode
	QueueLength  int
}

// EventListener, Player olaylarını dinleyen Observer fonksiyon tipidir.
type EventListener func(event PlayerEvent)
