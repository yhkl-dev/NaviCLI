package player

import (
	"github.com/wildeyedskies/go-mpv/mpv"
)

// Player defines the interface for audio playback operations.
// This abstraction allows NaviCLI to work with different media players.
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

	// SetVolume sets the volume level (0-100)
	SetVolume(volume float64) error

	// Seek skips by the given number of seconds (negative rewinds)
	Seek(seconds float64) error

	// IsPaused returns whether playback is currently paused
	IsPaused() (bool, error)

	// IsSongLoaded returns whether a song is currently loaded
	IsSongLoaded() (bool, error)

	// EventChannel returns a channel for receiving player events
	EventChannel() <-chan *mpv.Event

	// Cleanup performs cleanup operations (termination, resource release)
	Cleanup()
}

// PlayerConstants defines player state constants
const (
	PlayerError = iota + 3
)
