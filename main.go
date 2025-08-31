package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/mpvplayer"
	"github.com/yhkl-dev/NaviCLI/subsonic"
)

func formatDuration(seconds int) string {
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

type PlayerState struct {
	currentSong      *subsonic.Song
	currentSongIndex int
	isPlaying        bool
	isLoading        bool
	mux              sync.RWMutex
}

type Application struct {
	application    *tview.Application
	subsonicClient *subsonic.Client
	mpvInstance    *mpvplayer.Mpvplayer
	totalSongs     []subsonic.Song
	currentPage    int
	pageSize       int
	totalPages     int

	rootFlex    *tview.Flex
	songTable   *tview.Table
	statusBar   *tview.TextView
	progressBar *tview.TextView
	statsBar    *tview.TextView

	state *PlayerState
}

func (s *PlayerState) GetState() (song *subsonic.Song, index int, playing bool, loading bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.currentSong, s.currentSongIndex, s.isPlaying, s.isLoading
}

func (s *PlayerState) SetLoading(loading bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isLoading = loading
}

func (s *PlayerState) SetPlaying(playing bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isPlaying = playing
}

func (s *PlayerState) SetCurrentSong(song *subsonic.Song, index int) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.currentSong = song
	s.currentSongIndex = index
}

func (a *Application) setupPagination() {
	a.pageSize = 500
	a.currentPage = 1
	a.state = &PlayerState{
		currentSongIndex: -1,
		isLoading:        false,
		isPlaying:        false,
	}
}

func (a *Application) createStatusInfo(track subsonic.Song, index int, status, progressBar string) string {
	return fmt.Sprintf(`
[white]Current %d:
%s

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
		index+1, status, formatDuration(track.Duration),
		float64(track.Size)/1024/1024, track.Artist, track.Album, track.Album, progressBar)
}

func (a *Application) updateStatus(info string) {
	a.application.QueueUpdateDraw(func() {
		if a.statusBar != nil {
			a.statusBar.SetText(info)
		}
	})
}

func (a *Application) getPlayURL(trackID string) (string, bool) {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- ""
			}
		}()
		done <- a.subsonicClient.GetPlayURL(trackID)
	}()

	select {
	case url := <-done:
		return url, url != ""
	case <-time.After(10 * time.Second):
		return "", false
	}
}

func (a *Application) playSongAtIndex(index int) {
	if index < 0 || index >= len(a.totalSongs) {
		return
	}

	_, _, _, loading := a.state.GetState()
	if loading {
		return
	}

	currentTrack := a.totalSongs[index]
	a.state.SetLoading(true)
	a.state.SetCurrentSong(&currentTrack, index)
	a.state.SetPlaying(false)

	loadingStatus := fmt.Sprintf("[yellow]%s [darkgray](Loading...)", currentTrack.Title)
	loadingBar := "[yellow]▓▓▓[darkgray]░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ Loading..."
	a.updateStatus(a.createStatusInfo(currentTrack, index, loadingStatus, loadingBar))

	go func() {
		defer a.state.SetLoading(false)
		defer func() {
			if r := recover(); r != nil {
				a.state.SetPlaying(false)
				failedStatus := fmt.Sprintf("[red]%s [darkgray](Failed)", currentTrack.Title)
				a.updateStatus(a.createStatusInfo(currentTrack, index, failedStatus, "[red]Play Failed"))
			}
		}()

		playURL, ok := a.getPlayURL(currentTrack.ID)
		if !ok {
			return
		}

		if a.mpvInstance == nil {
			return
		}

		a.mpvInstance.Queue = []mpvplayer.QueueItem{{
			Id: currentTrack.ID, Uri: playURL, Title: currentTrack.Title,
			Artist: currentTrack.Artist, Duration: currentTrack.Duration,
		}}

		if a.mpvInstance.Mpv != nil {
			a.mpvInstance.Stop()
			time.Sleep(50 * time.Millisecond)
			a.mpvInstance.Play(playURL)
			a.state.SetPlaying(true)

			playingStatus := fmt.Sprintf("[lightgreen]%s", currentTrack.Title)
			playingBar := "[lightgreen]▓[darkgray]░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 0.0%"
			a.updateStatus(a.createStatusInfo(currentTrack, index, playingStatus, playingBar))

			time.Sleep(500 * time.Millisecond)
		}
	}()
}

func (a *Application) playNextSong() {
	if len(a.totalSongs) == 0 {
		return
	}

	_, currentIndex, _, loading := a.state.GetState()
	if loading {
		return
	}

	nextIndex := currentIndex + 1
	if nextIndex >= len(a.totalSongs) {
		nextIndex = 0
	}

	go a.playSongAtIndex(nextIndex)
}

func (a *Application) playPreviousSong() {
	if len(a.totalSongs) == 0 {
		return
	}

	_, currentIndex, _, loading := a.state.GetState()
	if loading {
		return
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(a.totalSongs) - 1
	}

	go a.playSongAtIndex(prevIndex)
}

func (a *Application) getCurrentPageData() []subsonic.Song {
	start := (a.currentPage - 1) * a.pageSize
	end := min(start+a.pageSize, len(a.totalSongs))
	return a.totalSongs[start:end]
}

func (a *Application) getPlaybackProgress() (currentPos, totalDuration float64, err error) {
	done := make(chan struct{})
	var hasError bool

	go func() {
		defer func() {
			if r := recover(); r != nil {
				hasError = true
			}
			close(done)
		}()

		pos, posErr := a.mpvInstance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
		if posErr != nil {
			hasError = true
			return
		}
		duration, durErr := a.mpvInstance.GetProperty("duration", mpv.FORMAT_DOUBLE)
		if durErr != nil {
			hasError = true
			return
		}
		currentPos = pos.(float64)
		totalDuration = duration.(float64)
	}()

	select {
	case <-done:
		if hasError {
			return 0, 0, fmt.Errorf("failed to get playback progress")
		}
		return currentPos, totalDuration, nil
	case <-time.After(200 * time.Millisecond):
		return 0, 0, fmt.Errorf("timeout getting playback progress")
	}
}

func (a *Application) createProgressBar(progress float64) string {
	const progressBarWidth = 30
	filledWidth := int(progress * float64(progressBarWidth))
	var bar string

	for i := range progressBarWidth {
		if i < filledWidth {
			bar += "[lightgreen]▓"
		} else {
			bar += "[darkgray]░"
		}
	}
	return bar + fmt.Sprintf("[white] %.1f%%", progress*100)
}

func (a *Application) updateIdleDisplay() {
	a.application.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			idleDisplay := `
[darkgray][about] [darkgray][credits] [darkgray][rss.xml]
[darkgray][patreon] [darkgray][podcasts.apple]
[darkgray][folder.jpg] [darkgray][enterprise mode]
[darkgray][invert] [darkgray][fullscreen]`
			a.progressBar.SetText(idleDisplay)
		}
	})
}

func (a *Application) updatePausedDisplay(song *subsonic.Song, index int) {
	a.application.QueueUpdateDraw(func() {
		if a.progressBar != nil && a.statusBar != nil {
			pausedDisplay := `
[darkgray]00:00:00 [darkgray][v-] [darkgray]100% [darkgray][v+] [darkgray][random]`
			a.progressBar.SetText(pausedDisplay)

			pausedStatus := fmt.Sprintf("[yellow]%s [darkgray](PAUSED)", song.Title)
			progressBar := "[darkgray]▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░ 0%"
			a.statusBar.SetText(a.createStatusInfo(*song, index, pausedStatus, progressBar))
		}
	})
}

func (a *Application) updatePlayingDisplay(song *subsonic.Song, index int, currentPos, totalDuration float64) {
	currentTime := formatDuration(int(currentPos))
	totalTime := formatDuration(int(totalDuration))
	
	progress := currentPos / totalDuration
	if progress > 1 {
		progress = 1
	} else if progress < 0 {
		progress = 0
	}

	progressText := fmt.Sprintf(`
[darkgray]%s/%s [darkgray][random]`, currentTime, totalTime)
	
	progressBar := a.createProgressBar(progress)
	playingStatus := fmt.Sprintf("[lightgreen]%s", song.Title)

	a.application.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(progressText)
		}
		if a.statusBar != nil {
			a.statusBar.SetText(a.createStatusInfo(*song, index, playingStatus, progressBar))
		}
	})
}

func (a *Application) updateProgressBar() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.application == nil || a.mpvInstance == nil {
				return
			}

			currentSong, currentIndex, isPlaying, isLoading := a.state.GetState()
			if isLoading {
				continue
			}

			if a.mpvInstance.Mpv == nil {
				a.updateIdleDisplay()
				continue
			}

			if !isPlaying {
				if currentSong != nil {
					a.updatePausedDisplay(currentSong, currentIndex)
				}
				continue
			}

			go func() {
				defer func() {
					if recover() != nil {
					}
				}()

				if a.mpvInstance == nil || a.mpvInstance.Mpv == nil {
					return
				}

				currentPos, totalDuration, err := a.getPlaybackProgress()
				if err != nil || totalDuration <= 0 || currentPos < 0 {
					return
				}

				song, index, _, _ := a.state.GetState()
				if song != nil {
					a.updatePlayingDisplay(song, index, currentPos, totalDuration)
				}
			}()

		case <-time.After(15 * time.Second):
			if a.application == nil {
				return
			}
		}
	}
}

func (a *Application) createHomepage() {
	a.progressBar = tview.NewTextView().
		SetDynamicColors(true)
	a.progressBar.SetBorder(false)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true)
	a.statusBar.SetBorder(false)

	a.statsBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	a.statsBar.SetBorder(false)

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	a.songTable.SetBorder(false)

	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Attributes(tcell.AttrBold)

	a.songTable.SetCell(0, 0, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 1, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 2, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 3, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 4, tview.NewTableCell("").
		SetStyle(headerStyle))

	a.songTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(a.totalSongs) {
			_, _, _, loading := a.state.GetState()
			if loading {
				return
			}
			go a.playSongAtIndex(row - 1)
		}
	})

	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.statusBar, 0, 1, false)

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.songTable, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftPanel, 0, 1, false).
		AddItem(rightPanel, 0, 2, true)

	a.rootFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.progressBar, 3, 0, false)

	a.application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case ' ':
				if a.mpvInstance == nil || a.mpvInstance.Mpv == nil {
					return nil
				}

				go func() {
					defer func() {
						if recover() != nil {
						}
					}()

					currentSong, currentIndex, isPlaying, _ := a.state.GetState()
					if currentSong == nil {
						return
					}

					a.mpvInstance.Pause()
					newPlayingState := !isPlaying
					a.state.SetPlaying(newPlayingState)

					var status, progressBar string
					if newPlayingState {
						status = fmt.Sprintf("[lightgreen]%s", currentSong.Title)
						progressBar = "[lightgreen]▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░ --%%"
					} else {
						status = fmt.Sprintf("[yellow]%s [darkgray](PAUSED)", currentSong.Title)
						progressBar = "[darkgray]▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░ --%%"
					}

					info := a.createStatusInfo(*currentSong, currentIndex, status, progressBar)
					a.updateStatus(info)
				}()
				return nil
			case 'n', 'N':
				a.playNextSong()
				return nil
			case 'p', 'P':
				a.playPreviousSong()
				return nil
			}
		}

		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			log.Println("user request exit program")

			if a.mpvInstance != nil && a.mpvInstance.Mpv != nil {
				a.mpvInstance.Command([]string{"quit"})
			}

			a.application.Stop()

			go func() {
				time.Sleep(1 * time.Second)
				os.Exit(0)
			}()
			return nil
		case tcell.KeyRight:
			a.playNextSong()
			return nil
		case tcell.KeyLeft:
			a.playPreviousSong()
			return nil
		}
		return event
	})
	a.application.SetRoot(a.rootFlex, true)

	welcomeMsg := fmt.Sprintf(`
[white]Current:
[lightgreen]Welcome to NaviCLI

[darkgray][play] Ready
[darkgray][source] Navidrome
[darkgray][favourite]

[gray]Press SPACE to play/pause
[gray]Press N/P or ←/→ for prev/next
[gray]Press ESC to exit
[gray]Select a track to start

[darkgray][red]func[darkgray] [green]navicli[darkgray]([yellow]task[darkgray] [lightblue]string[darkgray]) [lightblue]string[darkgray] {
[darkgray]    [red]return[darkgray] "^A series of mixes for listening while" [red]+[darkgray] task [red]+[darkgray] \
[darkgray]         "to focus the brain and i nspire the mind.[darkgray]"
[darkgray]}
[darkgray]
[darkgray]task := "[yellow]programming[darkgray]"

[darkgray]// %d songs
[darkgray]// Written by github.com/yhkl-dev
[darkgray]// Ready to play
[darkgray]// Auto-play next enabled`, len(a.totalSongs))
	a.statusBar.SetText(welcomeMsg)
}

func (a *Application) renderSongTable() {
	for i := a.songTable.GetRowCount() - 1; i > 0; i-- {
		a.songTable.RemoveRow(i)
	}
	pageData := a.getCurrentPageData()

	for i, song := range pageData {
		row := i + 1

		rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault)

		trackCell := tview.NewTableCell(fmt.Sprintf("%d:", row)).
			SetStyle(rowStyle.Foreground(tcell.ColorLightGreen)).
			SetAlign(tview.AlignRight)

		titleCell := tview.NewTableCell(song.Title).
			SetStyle(rowStyle.Foreground(tcell.ColorWhite)).
			SetExpansion(1)

		artistCell := tview.NewTableCell(song.Artist).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetMaxWidth(25)

		albumCell := tview.NewTableCell(song.Album).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetMaxWidth(25)

		durationCell := tview.NewTableCell(formatDuration(song.Duration)).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetAlign(tview.AlignRight)

		a.songTable.SetCell(row, 0, trackCell)
		a.songTable.SetCell(row, 1, titleCell)
		a.songTable.SetCell(row, 2, artistCell)
		a.songTable.SetCell(row, 3, albumCell)
		a.songTable.SetCell(row, 4, durationCell)
	}

	a.songTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))

	a.songTable.ScrollToBeginning()
}

func (a *Application) loadMusic() error {
	songs, err := a.subsonicClient.GetPlaylists()
	if err != nil {
		return fmt.Errorf("error get song list: %v", err)
	}

	if !reflect.DeepEqual(a.totalSongs, songs) {
		a.totalSongs = songs
		a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
		a.application.QueueUpdateDraw(func() {
			a.renderSongTable()
		})
	}
	return nil
}

func eventListener(ctx context.Context, m *mpv.Mpv) chan *mpv.Event {
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

func ViperInit() {
	required := []string{
		"server.url",
		"server.username",
		"server.password",
	}
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath(".")

	viper.SetDefault("keys.search", "/")

	if err := viper.ReadInConfig(); err != nil {
		os.Exit(1)
	}

	for _, key := range required {
		if !viper.IsSet(key) {

			os.Exit(1)
		}
	}
}

func main() {
	ViperInit()

	subsonicClient := subsonic.Init(
		viper.GetString("server.url"),
		viper.GetString("server.username"),
		viper.GetString("server.password"),
		"goplayer",
		"1.16.1",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mpvInstance, err := mpvplayer.CreateMPVInstance()
	if err != nil {
		log.Fatal(err)
	}

	app := &Application{
		application:    tview.NewApplication(),
		subsonicClient: subsonicClient,
		mpvInstance:    &mpvplayer.Mpvplayer{mpvInstance, eventListener(ctx, mpvInstance), make([]mpvplayer.QueueItem, 0), false},
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("receive exit signal, cleaning resource...")

		if app.mpvInstance != nil && app.mpvInstance.Mpv != nil {
			app.mpvInstance.Command([]string{"quit"})
			app.mpvInstance.TerminateDestroy()
		}

		cancel()
		app.application.Stop()

		go func() {
			time.Sleep(2 * time.Second)
			log.Println("force quit.")
			os.Exit(0)
		}()
	}()

	app.setupPagination()

	go func() {
		defer func() {
			if r := recover(); r != nil {

			}
		}()

		for {
			select {
			case event, ok := <-app.mpvInstance.EventChannel:
				if !ok {
					return
				}
				if event != nil && event.Event_Id == mpv.EVENT_END_FILE {
					app.application.QueueUpdateDraw(func() {
						app.playNextSong()
					})
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go app.updateProgressBar()
	go func() {
		if err := app.loadMusic(); err != nil {
			app.application.QueueUpdateDraw(func() {
				app.statusBar.SetText("[red]load music failed: " + err.Error())
			})
		}
	}()
	app.createHomepage()

	log.Println("start navicli...")
	err = app.application.Run()

	log.Println("program exiting, clear resource...")
	cancel()

	if app.mpvInstance != nil && app.mpvInstance.Mpv != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {

				}
			}()
			app.mpvInstance.Command([]string{"quit"})
			app.mpvInstance.TerminateDestroy()
		}()
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	log.Println("program exit.")
	os.Exit(0)
}
