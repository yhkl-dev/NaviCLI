package mpvplayer

import (
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
	return pos.(float64), err
}

func (m *Mpvplayer) GetDuration() (float64, error) {
	duration, err := m.GetProperty("duration", mpv.FORMAT_DOUBLE)
	return duration.(float64), err
}

func (m *Mpvplayer) Play(playURL string) {
	m.Command([]string{"loadfile", playURL})
}

func (m *Mpvplayer) Stop() error {
	return m.Command([]string{"stop"})
}

func (m *Mpvplayer) IsSongLoaded() (bool, error) {
	idle, err := m.GetProperty("idle-active", mpv.FORMAT_FLAG)
	return !idle.(bool), err
}

func (m *Mpvplayer) IsPaused() (bool, error) {
	pause, err := m.GetProperty("pause", mpv.FORMAT_FLAG)
	return pause.(bool), err
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
