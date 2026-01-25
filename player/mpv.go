package player

import (
	"context"
	"fmt"
	"time"

	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/domain"
	"github.com/yhkl-dev/NaviCLI/mpvplayer"
)

type MPVPlayer struct {
	instance *mpvplayer.Mpvplayer
}

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

func (p *MPVPlayer) Play(url string) error {
	if p.instance == nil || p.instance.Mpv == nil {
		return fmt.Errorf("MPV instance not initialized")
	}
	p.instance.Play(url)
	return nil
}

func (p *MPVPlayer) Pause() (int, error) {
	if p.instance == nil {
		return PlayerError, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.Pause()
}

func (p *MPVPlayer) Stop() error {
	if p.instance == nil || p.instance.Mpv == nil {
		return fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.Stop()
}

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

func (p *MPVPlayer) GetVolume() (float64, error) {
	if p.instance == nil {
		return 0, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.GetVolume()
}

func (p *MPVPlayer) SetVolume(volume float64) error {
	if p.instance == nil {
		return fmt.Errorf("MPV instance not initialized")
	}
	if volume < 0 {
		volume = 0
	} else if volume > 100 {
		volume = 100
	}
	return p.instance.Command([]string{"set", "volume", fmt.Sprintf("%.0f", volume)})
}

func (p *MPVPlayer) IsPlaying() bool {
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

func (p *MPVPlayer) IsPaused() (bool, error) {
	if p.instance == nil {
		return false, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.IsPaused()
}

func (p *MPVPlayer) IsSongLoaded() (bool, error) {
	if p.instance == nil {
		return false, fmt.Errorf("MPV instance not initialized")
	}
	return p.instance.IsSongLoaded()
}

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

func (p *MPVPlayer) ClearQueue() {
	if p.instance == nil {
		return
	}
	p.instance.Queue = make([]mpvplayer.QueueItem, 0)
}

func (p *MPVPlayer) EventChannel() <-chan *mpv.Event {
	if p.instance == nil {
		ch := make(chan *mpv.Event)
		close(ch)
		return ch
	}
	return p.instance.EventChannel
}

func (p *MPVPlayer) Cleanup() {
	if p.instance == nil || p.instance.Mpv == nil {
		return
	}

	defer func() {
		if recover() != nil {
		}
	}()

	time.Sleep(10 * time.Millisecond)

	p.instance.TerminateDestroy()
	p.instance.Mpv = nil
}

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
