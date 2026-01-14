package player

import (
	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/domain"
)

// Player defines the interface for audio playback operations
// This abstraction allows NaviCLI to work with different media players (MPV, VLC, etc.)
type Player interface {
	// Play starts playback of the given URL
	Play(url string) error

	// Pause toggles the pause state
	Pause() (int, error)

	// Stop stops playback completely
	Stop() error

	// GetProgress returns the current playback position and total duration
	GetProgress() (currentPos, totalDuration float64, err error)

	// GetVolume returns the current volume level
	GetVolume() (float64, error)

	// IsPlaying returns whether audio is currently playing
	IsPlaying() bool

	// IsPaused returns whether playback is currently paused
	IsPaused() (bool, error)

	// IsSongLoaded returns whether a song is currently loaded
	IsSongLoaded() (bool, error)

	// AddToQueue adds an item to the playback queue
	AddToQueue(item domain.QueueItem)

	// GetQueue returns the current playback queue
	GetQueue() []domain.QueueItem

	// ClearQueue clears the playback queue
	ClearQueue()

	// EventChannel returns a channel for receiving player events
	EventChannel() <-chan *mpv.Event

	// Cleanup performs cleanup operations (termination, resource release)
	Cleanup()
}

// PlayerConstants defines player state constants
const (
	PlayerStopped = iota
	PlayerPlaying
	PlayerPaused
	PlayerError
)
