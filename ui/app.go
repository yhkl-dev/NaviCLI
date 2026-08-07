package ui

import (
	"context"
	"fmt"
	"image"
	"log"
	"math/rand"
	"os"
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

const dataStartRow = 0

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

	rootFlex         *tview.Flex
	songTable        *tview.Table
	statusBar        *tview.TextView
	progressBar      *tview.TextView
	searchInput      *tview.InputField
	helpView         *HelpView
	isSearchMode     bool
	originalSongs    []domain.Song
	audioMonitor     *device.AudioMonitor
	tickCount        atomic.Int64
	serverConnected  atomic.Bool
	playingUpdateMu  sync.Mutex
	songsMu          sync.RWMutex
	cachedTermWidth  int
	lastWidthCheck   time.Time
	cachedTermHeight int
	sortMode         int
	repeatMode       int
	repeatMu         sync.Mutex
	leftTitleBar     *tview.TextView
	rightTitleBar    *tview.TextView

	coverMu           sync.RWMutex
	coverImg          image.Image
	coverArtID        string
	coverReq          uint64
	coverRendered     string
	coverRenderedID   string
	coverRenderedCols int
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

// Repeat modes.
const (
	repeatOff = iota
	repeatAll
	repeatOne
)

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
		repeatMode:  repeatAll,
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
	songs, err := a.library.GetAllSongs()
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			if a.statusBar != nil {
				a.statusBar.SetText("[" + tagDanger + "]Failed to load music: " + err.Error())
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
	a.refreshConnection(a.library.Ping() == nil)

	for {
		select {
		case <-ticker.C:
			a.refreshConnection(a.library.Ping() == nil)
		case <-a.ctx.Done():
			return
		}
	}
}

// refreshConnection stores the connection state and redraws the title bar
// when it changes, so the connection dot in the Now Playing header stays in
// sync without a full panel redraw.
func (a *App) refreshConnection(connected bool) {
	if a.serverConnected.Swap(connected) == connected {
		return
	}
	a.tviewApp.QueueUpdateDraw(a.updateSortTitle)
}

func (a *App) SortSongs() {
	a.songsMu.RLock()
	mode := sortModes[a.sortMode]
	a.songsMu.RUnlock()

	a.songsMu.Lock()
	defer a.songsMu.Unlock()
	if mode.less == nil {
		// Random: shuffle the whole list in place.
		rand.Shuffle(len(a.totalSongs), func(i, j int) {
			a.totalSongs[i], a.totalSongs[j] = a.totalSongs[j], a.totalSongs[i]
		})
		return
	}
	sort.SliceStable(a.totalSongs, func(i, j int) bool {
		return mode.less(a.totalSongs[i], a.totalSongs[j])
	})
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

func (a *App) repeatModeLocked() int {
	a.repeatMu.Lock()
	defer a.repeatMu.Unlock()
	return a.repeatMode
}

// cycleRepeat cycles repeat mode: off -> all -> one -> off.
func (a *App) cycleRepeat() {
	a.repeatMu.Lock()
	a.repeatMode = (a.repeatMode + 1) % 3
	a.repeatMu.Unlock()
	a.updateSortTitle()
}

func (a *App) updateSortTitle() {
	if a.leftTitleBar != nil {
		dot := fmt.Sprintf("[%s]●", tagDanger)
		if a.serverConnected.Load() {
			dot = fmt.Sprintf("[%s]●", tagAccent)
		}
		repeat := ""
		switch a.repeatModeLocked() {
		case repeatAll:
			repeat = "  [" + tagFgFaint + "]↻"
		case repeatOne:
			repeat = "  [" + tagFgFaint + "]↻1"
		}
		a.leftTitleBar.SetText(fmt.Sprintf("["+tagFg+"::b]Now playing[-:-:-]  %s%s", dot, repeat))
	}
	if a.rightTitleBar != nil {
		a.songsMu.RLock()
		count := len(a.totalSongs)
		mode := sortModes[a.sortMode].name
		a.songsMu.RUnlock()
		a.rightTitleBar.SetText(fmt.Sprintf(
			"["+tagFg+"::b]Library[-:-:-]  ["+tagFgDim+"]Page %d/%d · %d songs[-:-:-]  ["+tagFgFaint+"]%s",
			a.currentPage, a.totalPages, count, mode,
		))
	}
}

func (a *App) leftPanelTextWidth() int {
	w := a.getTerminalWidth() / 4
	if w < 24 {
		w = 24
	}
	return w - 4
}

// getTerminalSize returns the terminal width and height in one shot, cached for
// 1s. It tries `stty size` (one subprocess for both dimensions, reading the tty
// on stdin), falling back to `tput cols`/`tput lines`, then 80x24.
func (a *App) getTerminalSize() (int, int) {
	if time.Since(a.lastWidthCheck) < time.Second && a.cachedTermWidth > 0 {
		return a.cachedTermWidth, a.cachedTermHeight
	}
	a.lastWidthCheck = time.Now()

	w, h := a.cachedTermWidth, a.cachedTermHeight
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	if out, err := cmd.Output(); err == nil {
		var rows, cols int
		if _, err := fmt.Sscanf(string(out), "%d %d", &rows, &cols); err == nil && rows > 0 && cols > 0 {
			w, h = cols, rows
			a.cachedTermWidth, a.cachedTermHeight = w, h
			return w, h
		}
	}

	// Fallback: tput cols / tput lines (two subprocesses).
	if out, err := exec.Command("tput", "cols").Output(); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && v > 0 {
			w = v
		}
	}
	if out, err := exec.Command("tput", "lines").Output(); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && v > 0 {
			h = v
		}
	}

	a.cachedTermWidth, a.cachedTermHeight = w, h
	return w, h
}

func (a *App) getTerminalWidth() int {
	w, _ := a.getTerminalSize()
	return w
}

// getTerminalHeight returns the terminal height in rows. Used to cap the cover
// art height so the now-playing metadata stays visible on short terminals.
func (a *App) getTerminalHeight() int {
	_, h := a.getTerminalSize()
	return h
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
	a.loadCoverArt(&currentTrack)

	a.updateStatus(a.nowPlaying(currentTrack, ""))
	a.updateBottomBar(0, "0:00", FormatDuration(currentTrack.Duration), 0, "["+tagFgDim+"]Loading...", "◷")

	go func() {
		defer a.state.SetLoading(false)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("playSongAtIndex panic: %v", r)
				a.state.SetPlaying(false)
				a.updateStatus(a.nowPlaying(currentTrack, ""))
				a.updateBottomBar(0, "0:00", FormatDuration(currentTrack.Duration), 0, "["+tagDanger+"]Failed", "◷")
			}
		}()

		playURL, ok := a.getPlayURL(currentTrack.ID)
		if !ok {
			a.updateBottomBar(0, "0:00", FormatDuration(currentTrack.Duration), 0, "["+tagDanger+"]No stream URL", "◷")
			return
		}

		if err := a.player.Play(playURL); err != nil {
			a.updateBottomBar(0, "0:00", FormatDuration(currentTrack.Duration), 0, "["+tagDanger+"]Play failed", "◷")
			return
		}

		a.state.SetPlaying(true)

		a.updateStatus(a.nowPlaying(currentTrack, CreateTrackExtras(currentTrack)))
		a.updateBottomBar(0, "0:00", FormatDuration(currentTrack.Duration), 0, "["+tagAccent+"]▶ PLAYING", "◴")

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

	repeat := a.repeatModeLocked()
	if repeat == repeatOne {
		go a.playSongAtIndex(currentIndex)
		return
	}

	nextIndex := currentIndex + 1
	a.songsMu.RLock()
	n := len(a.totalSongs)
	a.songsMu.RUnlock()
	if nextIndex >= n {
		if repeat == repeatOff {
			return
		}
		nextIndex = 0
	}
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

// seekForward skips 10 seconds ahead.
func (a *App) seekForward() {
	if err := a.player.Seek(10); err != nil {
		log.Printf("seek forward: %v", err)
	}
}

// seekBackward rewinds 10 seconds.
func (a *App) seekBackward() {
	if err := a.player.Seek(-10); err != nil {
		log.Printf("seek backward: %v", err)
	}
}

// updateStatus updates the status bar
func (a *App) updateStatus(info string) {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.statusBar != nil {
			a.statusBar.SetText(info)
		}
	})
}

// updateBottomBar updates the bottom transport strip (progress, time, volume,
// status). This is the single home for live state.
func (a *App) updateBottomBar(progress float64, currentTime, totalTime string, volume float64, statusLabel, spinner string) {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(CreateBottomBar(progress, a.cfg.UI.ProgressBarWidth, currentTime, totalTime, volume, statusLabel, spinner))
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
		a.songTable.Select(0, 0)
	}
}

// previousPage moves to the previous page
func (a *App) previousPage() {
	if a.currentPage > 1 {
		a.currentPage--
		a.renderSongTable()
		a.updateStatusWithPageInfo()
		data := a.getCurrentPageData()
		if len(data) > 0 {
			a.songTable.Select(2*(len(data)-1), 0)
		}
	}
}

// updateStatusWithPageInfo refreshes the library title bar (page/sort info, via
// updateSortTitle) and, when idle, the left welcome panel. The page/sort info
// lives in the right title bar now, not appended to the welcome.
func (a *App) updateStatusWithPageInfo() {
	a.updateSortTitle()

	currentSong, _, isPlaying, _ := a.state.GetState()
	if currentSong != nil && isPlaying {
		// Keep current playing info in the left panel.
		return
	}

	a.songsMu.RLock()
	songCount := len(a.totalSongs)
	a.songsMu.RUnlock()
	if a.statusBar != nil {
		a.statusBar.SetText(CreateWelcomeMessage(songCount))
	}
}

// moveRowDown moves selection down in the song table
func (a *App) moveRowDown() {
	row, _ := a.songTable.GetSelection()
	data := a.getCurrentPageData()
	songIdx := row / 2
	if songIdx < len(data)-1 {
		a.songTable.Select(2*(songIdx+1), 0)
	} else if a.currentPage < a.totalPages {
		a.nextPage()
	}
}

// moveRowUp moves selection up in the song table
func (a *App) moveRowUp() {
	row, _ := a.songTable.GetSelection()
	songIdx := row / 2
	if songIdx > 0 {
		a.songTable.Select(2*(songIdx-1), 0)
	} else if a.currentPage > 1 {
		a.previousPage()
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

	currentPos, _, err := a.player.GetProgress()
	if err != nil || currentPos < 0 {
		currentPos = 0
	}

	// The song's metadata duration is the single source of truth for the
	// total, matching playSongAtIndex. The player's reported length can drift
	// by a fraction of a second and would make the total jump between paths.
	totalDuration := float64(currentSong.Duration)
	progress := 0.0
	if totalDuration > 0 {
		progress = currentPos / totalDuration
		if progress > 1 {
			progress = 1
		}
	}

	vol := 0.0
	if v, err := a.player.GetVolume(); err == nil {
		vol = v
	}

	a.updateStatus(a.nowPlaying(*currentSong, CreateTrackExtras(*currentSong)))
	a.updateBottomBar(progress, FormatDuration(int(currentPos)), FormatDuration(currentSong.Duration), vol, "["+tagWarn+"]⏸ PAUSED", "◷")
}
