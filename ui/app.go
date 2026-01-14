package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/config"
	"github.com/yhkl-dev/NaviCLI/coverart"
	"github.com/yhkl-dev/NaviCLI/domain"
	"github.com/yhkl-dev/NaviCLI/library"
	"github.com/yhkl-dev/NaviCLI/player"
)

// App represents the TUI application
type App struct {
	tviewApp   *tview.Application
	cfg        *config.Config
	library    library.Library
	player     player.Player
	ctx        context.Context
	totalSongs []domain.Song
	state      *domain.PlayerState

	currentPage int
	pageSize    int
	totalPages  int

	rootFlex       *tview.Flex
	songTable      *tview.Table
	statusBar      *tview.TextView
	progressBar    *tview.TextView
	searchView     *SearchView
	helpView       *HelpView
	queueView      *QueueView
	coverConverter *coverart.Converter
	currentCover   string // 缓存当前封面 ASCII
}

// NewApp creates a new TUI application with dependency injection
func NewApp(ctx context.Context, cfg *config.Config, lib library.Library, plr player.Player) *App {
	return &App{
		tviewApp:       tview.NewApplication(),
		cfg:            cfg,
		library:        lib,
		player:         plr,
		ctx:            ctx,
		state:          domain.NewPlayerState(),
		pageSize:       cfg.UI.PageSize,
		currentPage:    1,
		coverConverter: coverart.NewConverter(),
	}
}

// Run starts the application
func (a *App) Run() error {
	a.createHomepage()
	go a.updateProgressBar()
	go a.loadMusic()
	go a.handlePlayerEvents()
	go a.handleTerminalResize()

	log.Println("start navicli...")
	return a.tviewApp.Run()
}

// Stop stops the application
func (a *App) Stop() {
	if a.tviewApp != nil {
		a.tviewApp.Stop()
	}
}

// loadMusic loads songs from the library
func (a *App) loadMusic() {
	// Load enough songs for multiple pages (10x page size for better experience)
	loadSize := a.cfg.UI.PageSize * 10
	songs, err := a.library.GetRandomSongs(loadSize)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			if a.statusBar != nil {
				a.statusBar.SetText("[red]Failed to load music: " + err.Error())
			}
		})
		return
	}

	domainSongs := songs
	if !reflect.DeepEqual(a.totalSongs, domainSongs) {
		a.totalSongs = domainSongs
		a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
		a.currentPage = 1 // Reset to first page
		a.tviewApp.QueueUpdateDraw(func() {
			a.renderSongTable()
			a.updateStatusWithPageInfo()
		})
	}
}

// handlePlayerEvents handles MPV player events
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

// handleTerminalResize handles terminal resize events
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
					if len(a.totalSongs) > 0 {
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

// getTerminalWidth returns the current terminal width
func (a *App) getTerminalWidth() int {
	cmd := exec.Command("tput", "cols")
	output, err := cmd.Output()
	if err != nil {
		return 80
	}
	width, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// playSongAtIndex plays a song at the specified index
func (a *App) playSongAtIndex(index int) {
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
	a.updateStatus(FormatSongInfo(currentTrack, index, loadingStatus, loadingBar))

	go func() {
		defer a.state.SetLoading(false)
		defer func() {
			if r := recover(); r != nil {
				a.state.SetPlaying(false)
				failedStatus := fmt.Sprintf("[red]%s [darkgray](Failed)", currentTrack.Title)
				a.updateStatus(FormatSongInfo(currentTrack, index, failedStatus, "[red]Play Failed"))
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

		playingStatus := fmt.Sprintf("[lightgreen]%s", currentTrack.Title)
		playingBar := "[lightgreen]▓[darkgray]░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 0.0%"
		a.updateStatus(FormatSongInfo(currentTrack, index, playingStatus, playingBar))

		// Load and display cover art asynchronously
		go a.loadCoverArt(currentTrack)

		time.Sleep(500 * time.Millisecond)
	}()
}

// getPlayURL retrieves the play URL with timeout
func (a *App) getPlayURL(trackID string) (string, bool) {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- ""
			}
		}()
		done <- a.library.GetPlayURL(trackID)
	}()

	select {
	case url := <-done:
		return url, url != ""
	case <-time.After(10 * time.Second):
		return "", false
	}
}

// loadCoverArt loads and displays cover art for the current song
func (a *App) loadCoverArt(song domain.Song) {
	if song.CoverArt == "" {
		return
	}

	coverURL := a.library.GetCoverArtURL(song.CoverArt)
	ascii, err := a.coverConverter.ConvertFromURL(coverURL)
	if err != nil {
		log.Printf("Failed to load cover art: %v", err)
		ascii, _ = a.coverConverter.ConvertFromURL("") // Use placeholder
	}

	// 缓存封面并更新状态栏
	a.currentCover = ascii

	// 重新格式化当前播放信息以包含封面
	currentSong, index, isPlaying, _ := a.state.GetState()
	if currentSong != nil && isPlaying {
		playingStatus := fmt.Sprintf("[lightgreen]%s", currentSong.Title)
		playingBar := "[lightgreen]▓[darkgray]░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 0.0%"
		a.updateStatus(FormatSongInfoWithCover(*currentSong, index, playingStatus, playingBar, ascii))
	}
}

// playNextSong plays the next song in the list
func (a *App) playNextSong() {
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

// playPreviousSong plays the previous song in the list
func (a *App) playPreviousSong() {
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
	pageInfo := fmt.Sprintf("[gray]Page %d/%d | %d songs total",
		a.currentPage, a.totalPages, len(a.totalSongs))

	currentSong, _, isPlaying, _ := a.state.GetState()
	if currentSong != nil && isPlaying {
		// Keep current playing info if song is playing
		return
	}

	a.statusBar.SetText(CreateWelcomeMessage(len(a.totalSongs)) + "\n\n" + pageInfo)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
