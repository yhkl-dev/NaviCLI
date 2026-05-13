package mpvplayer

import (
	"fmt"

	"github.com/wildeyedskies/go-mpv/mpv"
)

const (
	PlayerStopped = iota
	PlayerPlaying
	PlayerPaused
	PlayerError
)

type QueueItem struct {
	Id       string
	Uri      string
	Title    string
	Artist   string
	Duration int
}

type Mpvplayer struct {
	*mpv.Mpv
	EventChannel      chan *mpv.Event
	Queue             []QueueItem
	ReplaceInProgress bool
}

func (m *Mpvplayer) GetProgress() (float64, error) {
	pos, err := m.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
	if err != nil {
		return 0, err
	}
	val, ok := pos.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for time-pos: %T", pos)
	}
	return val, nil
}

func (m *Mpvplayer) GetDuration() (float64, error) {
	duration, err := m.GetProperty("duration", mpv.FORMAT_DOUBLE)
	if err != nil {
		return 0, err
	}
	val, ok := duration.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for duration: %T", duration)
	}
	return val, nil
}

func (m *Mpvplayer) GetVolume() (float64, error) {
	volume, err := m.GetProperty("volume", mpv.FORMAT_DOUBLE)
	if err != nil {
		return 0, err
	}
	val, ok := volume.(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for volume: %T", volume)
	}
	return val, nil
}

func (m *Mpvplayer) Play(playURL string) {
	m.Command([]string{"loadfile", playURL})
}

func (m *Mpvplayer) Stop() error {
	return m.Command([]string{"stop"})
}

func (m *Mpvplayer) IsSongLoaded() (bool, error) {
	idle, err := m.GetProperty("idle-active", mpv.FORMAT_FLAG)
	if err != nil {
		return false, err
	}
	val, ok := idle.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type for idle-active: %T", idle)
	}
	return !val, nil
}

func (m *Mpvplayer) IsPaused() (bool, error) {
	pause, err := m.GetProperty("pause", mpv.FORMAT_FLAG)
	if err != nil {
		return false, err
	}
	val, ok := pause.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type for pause: %T", pause)
	}
	return val, nil
}

func (m *Mpvplayer) Pause() (int, error) {
	loaded, err := m.IsSongLoaded()
	if err != nil {
		return PlayerError, err
	}
	pause, err := m.IsPaused()
	if err != nil {
		return PlayerError, err
	}

	if loaded {
		err := m.Command([]string{"cycle", "pause"})
		if err != nil {
			return PlayerError, err
		}
		if pause {
			return PlayerPlaying, nil
		}
		return PlayerPaused, nil
	} else {
		if len(m.Queue) != 0 {
			err := m.Command([]string{"loadfile", m.Queue[0].Uri})
			return PlayerPlaying, err
		} else {
			return PlayerStopped, nil
		}
	}
}

func CreateMPVInstance() (*mpv.Mpv, error) {
	mpvInstance := mpv.Create()

	mpvInstance.SetOptionString("audio-display", "no")
	mpvInstance.SetOptionString("video", "no")
	mpvInstance.ObserveProperty(0, "cache-buffering-state", mpv.FORMAT_INT64)
	mpvInstance.ObserveProperty(0, "demuxer-cache-duration", mpv.FORMAT_INT64)

	err := mpvInstance.Initialize()
	if err != nil {
		mpvInstance.TerminateDestroy()
		return nil, err
	}
	return mpvInstance, nil
}
