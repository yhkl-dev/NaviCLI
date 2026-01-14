package player

import (
	"context"
	"fmt"
	"time"

	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/domain"
	"github.com/yhkl-dev/NaviCLI/mpvplayer"
)

// MPVPlayer implements the Player interface using MPV media player
type MPVPlayer struct {
	instance *mpvplayer.Mpvplayer
}

// NewMPVPlayer creates a new MPVPlayer instance
func NewMPVPlayer(ctx context.Context) (*MPVPlayer, error) {
	mpvInstance, err := mpvplayer.CreateMPVInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create MPV instance: %w", err)
	}

	player := &MPVPlayer{
		instance: &mpvplayer.Mpvplayer{
			Mpv:               mpvInstance,
			EventChannel:      createEventListener(ctx, mpvInstance),
			Queue:             make([]mpvplayer.QueueItem, 0),
			ReplaceInProgress: false,
		},
	}

	return player, nil
}

// Play starts playback of the given URL
func (p *MPVPlayer) Play(url string) error {
	if p.instance == nil || p.instance.Mpv == nil {
		return fmt.Errorf("MPV instance not initialized")
	}
	p.instance.Play(url)
	return nil
}

// Pause toggles the pause state
func (p *MPVPlayer) Pause() (int, error) {
	if p.instance == nil {
		return PlayerError, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.Pause()
}

// Stop stops playback
func (p *MPVPlayer) Stop() error {
	if p.instance == nil || p.instance.Mpv == nil {
		return fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.Stop()
}

// GetProgress returns the current playback position and total duration
func (p *MPVPlayer) GetProgress() (currentPos, totalDuration float64, err error) {
	if p.instance == nil || p.instance.Mpv == nil {
		return 0, 0, fmt.Errorf("MPV instance not initialized")
	}

	pos, err := p.instance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
	if err != nil {
		return 0, 0, err
	}
	duration, err := p.instance.GetProperty("duration", mpv.FORMAT_DOUBLE)
	if err != nil {
		return 0, 0, err
	}

	return pos.(float64), duration.(float64), nil
}

// GetVolume returns the current volume level
func (p *MPVPlayer) GetVolume() (float64, error) {
	if p.instance == nil {
		return 0, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.GetVolume()
}

// IsPlaying returns whether audio is currently playing (not implemented in base mpvplayer)
func (p *MPVPlayer) IsPlaying() bool {
	// This would need to check the actual MPV state
	// For now, delegate to IsPaused
	paused, err := p.IsPaused()
	if err != nil {
		return false
	}
	loaded, err := p.IsSongLoaded()
	if err != nil {
		return false
	}
	return loaded && !paused
}

// IsPaused returns whether playback is paused
func (p *MPVPlayer) IsPaused() (bool, error) {
	if p.instance == nil {
		return false, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.IsPaused()
}

// IsSongLoaded returns whether a song is loaded
func (p *MPVPlayer) IsSongLoaded() (bool, error) {
	if p.instance == nil {
		return false, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.IsSongLoaded()
}

// AddToQueue adds an item to the playback queue
func (p *MPVPlayer) AddToQueue(item domain.QueueItem) {
	if p.instance == nil {
		return
	}
	p.instance.Queue = append(p.instance.Queue, mpvplayer.QueueItem{
		Id:       item.ID,
		Uri:      item.URI,
		Title:    item.Title,
		Artist:   item.Artist,
		Duration: item.Duration,
	})
}

// GetQueue returns the current playback queue
func (p *MPVPlayer) GetQueue() []domain.QueueItem {
	if p.instance == nil {
		return []domain.QueueItem{}
	}

	queue := make([]domain.QueueItem, len(p.instance.Queue))
	for i, item := range p.instance.Queue {
		queue[i] = domain.QueueItem{
			ID:       item.Id,
			URI:      item.Uri,
			Title:    item.Title,
			Artist:   item.Artist,
			Duration: item.Duration,
		}
	}
	return queue
}

// ClearQueue clears the playback queue
func (p *MPVPlayer) ClearQueue() {
	if p.instance == nil {
		return
	}
	p.instance.Queue = make([]mpvplayer.QueueItem, 0)
}

// EventChannel returns the event channel
func (p *MPVPlayer) EventChannel() <-chan *mpv.Event {
	if p.instance == nil {
		ch := make(chan *mpv.Event)
		close(ch)
		return ch
	}
	return p.instance.EventChannel
}

// Cleanup performs cleanup operations
func (p *MPVPlayer) Cleanup() {
	if p.instance != nil && p.instance.Mpv != nil {
		p.instance.Command([]string{"quit"})
		p.instance.TerminateDestroy()
	}
}

// createEventListener creates an event listener for MPV events
func createEventListener(ctx context.Context, m *mpv.Mpv) chan *mpv.Event {
	c := make(chan *mpv.Event)
	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				e := m.WaitEvent(1)
				if e == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				select {
				case c <- e:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return c
}
