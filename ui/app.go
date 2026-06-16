package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rivo/tview"
	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/config"
	"github.com/yhkl-dev/NaviCLI/device"
	"github.com/yhkl-dev/NaviCLI/domain"
	"github.com/yhkl-dev/NaviCLI/library"
	"github.com/yhkl-dev/NaviCLI/player"
)

const dataStartRow = 1

type App struct {
	tviewApp    *tview.Application
	cfg         *config.Config
	library     library.Library
	player      player.Player
	ctx         context.Context
	totalSongs  []domain.Song
	state       *domain.PlayerState
	keyBindings *KeyBindingManager

	currentPage int
	pageSize    int
	totalPages  int

	rootFlex      *tview.Flex
	songTable     *tview.Table
	statusBar     *tview.TextView
	progressBar   *tview.TextView
	searchInput   *tview.InputField
	helpView      *HelpView
	queueView     *QueueView
	isSearchMode  bool
	originalSongs []domain.Song
	audioMonitor     *device.AudioMonitor
	tickCount        atomic.Int64
	serverConnected  atomic.Bool
	playingUpdateMu  sync.Mutex
	songsMu          sync.RWMutex
	cachedTermWidth  int
	lastWidthCheck   time.Time
	sortMode         int
	songSource       int // 0=getRandomSongs, 1=getAlbumList2
	leftTitleBar     *tview.TextView
	rightTitleBar    *tview.TextView
}

var songSources = []struct {
	name string
}{
	{"Random"},
	{"Albums"},
}

var sortModes = []struct {
	name string
	less func(a, b domain.Song) bool
}{
	{"Random", nil},
	{"Title", func(a, b domain.Song) bool { return a.Title < b.Title }},
	{"Artist", func(a, b domain.Song) bool { return a.Artist < b.Artist }},
	{"Album", func(a, b domain.Song) bool {
		if a.Album != b.Album {
			return a.Album < b.Album
		}
		return a.Track < b.Track
	}},
}

func NewApp(ctx context.Context, cfg *config.Config, lib library.Library, plr player.Player) *App {
	return &App{
		tviewApp:    tview.NewApplication(),
		cfg:         cfg,
		library:     lib,
		player:      plr,
		ctx:         ctx,
		state:       domain.NewPlayerState(),
		keyBindings: NewKeyBindingManager(),
		pageSize:    cfg.UI.PageSize,
		currentPage: 1,
		sortMode:    1, // default: Title
	}
}

func (a *App) Run() error {
	a.createHomepage()
	go a.updateProgressBar()
	go a.loadMusic()
	go a.monitorConnection()
	go a.handlePlayerEvents()
	go a.handleTerminalResize()
	a.startAudioMonitor()

	log.Println("start navicli...")
	return a.tviewApp.Run()
}

func (a *App) Stop() {
	if a.tviewApp != nil {
		a.tviewApp.Stop()
	}
}

func (a *App) loadMusic() {
	fetchSize := a.cfg.UI.FetchSize
	var songs []domain.Song
	var err error

	a.songsMu.RLock()
	src := a.songSource
	a.songsMu.RUnlock()
	if src == 0 {
		songs, err = a.library.GetRandomSongs(fetchSize)
	} else {
		songs, err = a.library.GetAlbumSongs("alphabeticalByName")
	}

	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			if a.statusBar != nil {
				a.statusBar.SetText("[red]Failed to load music: " + err.Error())
			}
		})
		return
	}

	a.songsMu.Lock()
	a.totalSongs = songs
	a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
	if a.totalPages == 0 {
		a.totalPages = 1
	}
	a.currentPage = 1
	a.songsMu.Unlock()

	a.SortSongs()

	a.tviewApp.QueueUpdateDraw(func() {
		a.renderSongTable()
		a.updateStatusWithPageInfo()
	})
}

func (a *App) handlePlayerEvents() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Player event handler panic recovered: %v", r)
		}
	}()

	eventChan := a.player.EventChannel()
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			if event != nil && event.Event_Id == mpv.EVENT_END_FILE {
				a.tviewApp.QueueUpdateDraw(func() {
					a.playNextSong()
				})
			}
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *App) handleTerminalResize() {
	lastWidth := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.tviewApp == nil {
				return
			}
			currentWidth := a.getTerminalWidth()
			if currentWidth != lastWidth && lastWidth != 0 {
				a.tviewApp.QueueUpdateDraw(func() {
					a.songsMu.RLock()
					hasSongs := len(a.totalSongs) > 0
					a.songsMu.RUnlock()
					if hasSongs {
						a.renderSongTable()
					}
				})
			}
			lastWidth = currentWidth
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *App) monitorConnection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial ping
	a.serverConnected.Store(a.library.Ping() == nil)

	for {
		select {
		case <-ticker.C:
			a.serverConnected.Store(a.library.Ping() == nil)
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *App) SortSongs() {
	a.songsMu.RLock()
	mode := sortModes[a.sortMode]
	a.songsMu.RUnlock()
	if mode.less == nil {
		return
	}
	a.songsMu.Lock()
	sort.SliceStable(a.totalSongs, func(i, j int) bool {
		return mode.less(a.totalSongs[i], a.totalSongs[j])
	})
	a.songsMu.Unlock()
}

func (a *App) cycleSortMode() {
	a.songsMu.Lock()
	a.sortMode = (a.sortMode + 1) % len(sortModes)
	a.songsMu.Unlock()
	a.SortSongs()
	a.renderSongTable()
	a.updateStatusWithPageInfo()
	a.updateSortTitle()
}

func (a *App) cycleSongSource() {
	a.songsMu.Lock()
	a.songSource = (a.songSource + 1) % len(songSources)
	a.songsMu.Unlock()
	go a.loadMusic()
	a.updateSortTitle()
}

func (a *App) updateSortTitle() {
	a.songsMu.RLock()
	mode := sortModes[a.sortMode]
	src := songSources[a.songSource]
	a.songsMu.RUnlock()
	if a.rightTitleBar != nil {
		a.rightTitleBar.SetText(fmt.Sprintf("[#ffb300]── Library  [darkgray][%s · %s]", src.name, mode.name))
	}
	if a.leftTitleBar != nil {
		a.leftTitleBar.SetText(fmt.Sprintf("[#ffb300]── Now Playing  [darkgray][%s]", mode.name))
	}
}

func (a *App) leftPanelTextWidth() int {
	w := a.getTerminalWidth() / 4
	if w < 24 {
		w = 24
	}
	return w - 4
}

func (a *App) getTerminalWidth() int {
	if time.Since(a.lastWidthCheck) < time.Second && a.cachedTermWidth > 0 {
		return a.cachedTermWidth
	}
	a.lastWidthCheck = time.Now()

	cmd := exec.Command("tput", "cols")
	output, err := cmd.Output()
	if err != nil {
		if a.cachedTermWidth > 0 {
			return a.cachedTermWidth
		}
		return 80
	}
	width, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil || width <= 0 {
		if a.cachedTermWidth > 0 {
			return a.cachedTermWidth
		}
		return 80
	}
	a.cachedTermWidth = width
	return width
}

func (a *App) playSongAtIndex(index int) {
	a.songsMu.RLock()
	if index < 0 || index >= len(a.totalSongs) {
		a.songsMu.RUnlock()
		return
	}
	currentTrack := a.totalSongs[index]
	a.songsMu.RUnlock()

	_, _, _, loading := a.state.GetState()
	if loading {
		return
	}
	a.state.SetLoading(true)
	a.state.SetCurrentSong(&currentTrack, index)
	a.state.SetPlaying(false)

	loadingStatus := fmt.Sprintf("[#ffb300]%s [darkgray](Loading...)", currentTrack.Title)
	a.updateStatus(FormatSongInfo(currentTrack, loadingStatus, "", "[darkgray]Vol: [...", a.leftPanelTextWidth(), a.serverConnected.Load(), ""))

	go func() {
		defer a.state.SetLoading(false)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("playSongAtIndex panic: %v", r)
				a.state.SetPlaying(false)
				failedStatus := fmt.Sprintf("[red]%s [darkgray](Failed)", currentTrack.Title)
				a.updateStatus(FormatSongInfo(currentTrack, failedStatus, "", "[darkgray]Vol: [...", a.leftPanelTextWidth(), a.serverConnected.Load(), ""))
			}
		}()

		playURL, ok := a.getPlayURL(currentTrack.ID)
		if !ok {
			return
		}

		if err := a.player.Play(playURL); err != nil {
			return
		}

		a.player.AddToQueue(domain.QueueItem{
			ID:       currentTrack.ID,
			URI:      playURL,
			Title:    currentTrack.Title,
			Artist:   currentTrack.Artist,
			Duration: currentTrack.Duration,
		})

		a.state.SetPlaying(true)

		playingStatus := fmt.Sprintf("[#ffb300]▶ PLAYING")
		a.updateStatus(FormatSongInfo(currentTrack, playingStatus, "◴", "[darkgray]Vol: [...", a.leftPanelTextWidth(), a.serverConnected.Load(), CreatePlayingExtras(currentTrack, a.leftPanelTextWidth())))

		a.tviewApp.QueueUpdateDraw(func() {
			a.renderSongTable()
		})
	}()
}

func (a *App) getPlayURL(trackID string) (string, bool) {
	url := a.library.GetPlayURL(trackID)
	return url, url != ""
}

// playNextSong plays the next song in the list
func (a *App) playNextSong() {
	a.songsMu.RLock()
	if len(a.totalSongs) == 0 {
		a.songsMu.RUnlock()
		return
	}
	a.songsMu.RUnlock()

	_, currentIndex, _, loading := a.state.GetState()
	if loading {
		return
	}

	nextIndex := currentIndex + 1
	a.songsMu.RLock()
	if nextIndex >= len(a.totalSongs) {
		nextIndex = 0
	}
	a.songsMu.RUnlock()

	go a.playSongAtIndex(nextIndex)
}

// playPreviousSong plays the previous song in the list
func (a *App) playPreviousSong() {
	a.songsMu.RLock()
	if len(a.totalSongs) == 0 {
		a.songsMu.RUnlock()
		return
	}
	a.songsMu.RUnlock()

	_, currentIndex, _, loading := a.state.GetState()
	if loading {
		return
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		a.songsMu.RLock()
		prevIndex = len(a.totalSongs) - 1
		a.songsMu.RUnlock()
	}

	go a.playSongAtIndex(prevIndex)
}

// volumeUp increases the volume by 5%
func (a *App) volumeUp() {
	currentVol, err := a.player.GetVolume()
	if err != nil {
		return
	}
	newVol := currentVol + 5
	if newVol > 100 {
		newVol = 100
	}
	a.player.SetVolume(newVol)
}

// volumeDown decreases the volume by 5%
func (a *App) volumeDown() {
	currentVol, err := a.player.GetVolume()
	if err != nil {
		return
	}
	newVol := currentVol - 5
	if newVol < 0 {
		newVol = 0
	}
	a.player.SetVolume(newVol)
}

// updateStatus updates the status bar
func (a *App) updateStatus(info string) {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.statusBar != nil {
			a.statusBar.SetText(info)
		}
	})
}

// getCurrentPageData returns songs for the current page
func (a *App) getCurrentPageData() []domain.Song {
	a.songsMu.RLock()
	defer a.songsMu.RUnlock()
	start := (a.currentPage - 1) * a.pageSize
	end := min(start+a.pageSize, len(a.totalSongs))
	return a.totalSongs[start:end]
}

// nextPage moves to the next page
func (a *App) nextPage() {
	if a.currentPage < a.totalPages {
		a.currentPage++
		a.renderSongTable()
		a.updateStatusWithPageInfo()
	}
}

// previousPage moves to the previous page
func (a *App) previousPage() {
	if a.currentPage > 1 {
		a.currentPage--
		a.renderSongTable()
		a.updateStatusWithPageInfo()
	}
}

// updateStatusWithPageInfo updates status bar with page information
func (a *App) updateStatusWithPageInfo() {
	a.songsMu.RLock()
	songCount := len(a.totalSongs)
	a.songsMu.RUnlock()
	pageInfo := fmt.Sprintf("[gray]Page %d/%d | %d songs total",
		a.currentPage, a.totalPages, songCount)

	currentSong, _, isPlaying, _ := a.state.GetState()
	if currentSong != nil && isPlaying {
		// Keep current playing info if song is playing
		return
	}

	if a.statusBar != nil {
		a.statusBar.SetText(CreateWelcomeMessage(len(a.totalSongs)) + "\n\n" + pageInfo)
	}
}

// moveRowDown moves selection down in the song table
func (a *App) moveRowDown() {
	row, _ := a.songTable.GetSelection()
	data := a.getCurrentPageData()
	if row < len(data)-1 {
		a.songTable.Select(row+1, 0)
	} else if row == len(data)-1 && a.currentPage < a.totalPages {
		// Auto-move to next page when reaching end of current page
		a.nextPage()
		a.songTable.Select(0, 0)
	}
}

// moveRowUp moves selection up in the song table
func (a *App) moveRowUp() {
	row, _ := a.songTable.GetSelection()
	if row > 0 {
		a.songTable.Select(row-1, 0)
	} else if row == 0 && a.currentPage > 1 {
		a.previousPage()
		data := a.getCurrentPageData()
		if len(data) > 0 {
			a.songTable.Select(len(data)-1, 0)
		}
	}
}

func (a *App) goToFirstPage() {
	if a.currentPage != 1 {
		a.currentPage = 1
		a.renderSongTable()
		a.updateStatusWithPageInfo()
		a.songTable.Select(dataStartRow, 0)
	}
}

func (a *App) goToLastPage() {
	if a.currentPage != a.totalPages {
		a.currentPage = a.totalPages
		a.renderSongTable()
		a.updateStatusWithPageInfo()
		a.songTable.Select(dataStartRow, 0)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *App) startAudioMonitor() {
	a.audioMonitor = device.NewAudioMonitor(a.ctx, func() {
		a.onAudioDeviceDisconnected()
	})
	a.audioMonitor.Start()
	log.Println("Audio device monitor started")
}

func (a *App) onAudioDeviceDisconnected() {
	_, _, isPlaying, _ := a.state.GetState()
	if !isPlaying {
		return
	}

	loaded, err := a.player.IsSongLoaded()
	if err != nil || !loaded {
		return
	}

	_, err = a.player.Pause()
	if err != nil {
		log.Printf("Failed to pause on device disconnect: %v", err)
		return
	}

	a.state.SetPlaying(false)

	currentSong, _, _, _ := a.state.GetState()
	if currentSong == nil {
		return
	}

	pausedStatus := fmt.Sprintf("[#ff9800]⏸ PAUSED")
	statusText := FormatSongInfo(*currentSong, pausedStatus, "◷", "[darkgray]Vol: [...", a.leftPanelTextWidth(), a.serverConnected.Load(), "")

	a.tviewApp.QueueUpdateDraw(func() {
		if a.statusBar != nil {
			a.statusBar.SetText(statusText)
		}
	})
}
