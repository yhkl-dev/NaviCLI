package ui

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yhkl-dev/NaviCLI/domain"
)

func (a *App) createHomepage() {
	a.progressBar = tview.NewTextView().
		SetDynamicColors(true)
	a.progressBar.SetBorder(false)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true)
	a.statusBar.SetBorder(false)

	a.searchInput = tview.NewInputField().
		SetLabel("[#ffb300]Search: ").
		SetFieldWidth(0).
		SetPlaceholder("Type to search, ESC to clear, ENTER to filter...").
		SetFieldBackgroundColor(tcell.ColorDefault)
	a.searchInput.SetBorder(false)

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.songTable.SetBorder(false)

	a.helpView = NewHelpView(a)
	a.queueView = NewQueueView(a)

	a.setupTableHeaders()
	a.setupSearchInput()
	a.setupInputHandlers()

	a.leftTitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#ffb300]── Now Playing ")

	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.leftTitleBar, 1, 0, false).
		AddItem(a.statusBar, 0, 1, false)

	a.rightTitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#ffb300]── Library ")

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.rightTitleBar, 1, 0, false).
		AddItem(a.searchInput, 1, 0, false).
		AddItem(a.songTable, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftPanel, 0, 1, false).
		AddItem(rightPanel, 0, 3, true)

	a.rootFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.progressBar, 3, 0, false)

	a.tviewApp.SetRoot(a.rootFlex, true)
}

func (a *App) setupTableHeaders() {
	headerStyle := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffb300)).Attributes(tcell.AttrBold)

	a.songTable.SetCell(0, 0, tview.NewTableCell("#").SetStyle(headerStyle).SetAlign(tview.AlignRight))
	a.songTable.SetCell(0, 1, tview.NewTableCell("Title").SetStyle(headerStyle).SetExpansion(1))
	a.songTable.SetCell(0, 2, tview.NewTableCell("Duration").SetStyle(headerStyle).SetAlign(tview.AlignRight))
	a.songTable.SetCell(0, 3, tview.NewTableCell("Artist").SetStyle(headerStyle))
	a.songTable.SetCell(0, 4, tview.NewTableCell("Album").SetStyle(headerStyle))
}

func (a *App) setupSearchInput() {
	a.searchInput.SetChangedFunc(func(text string) {
		if text == "" && a.isSearchMode {
			a.clearSearch()
		}
	})

	a.searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := a.searchInput.GetText()
			if query != "" {
				a.performSearch(query)
			}
		} else if key == tcell.KeyEscape {
			a.clearSearch()
			a.tviewApp.SetFocus(a.songTable)
		}
	})

	a.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.clearSearch()
			a.tviewApp.SetFocus(a.songTable)
			return nil
		}
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyTab {
			a.tviewApp.SetFocus(a.songTable)
			return nil
		}
		return event
	})
}

func (a *App) setupInputHandlers() {
	a.songTable.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			_, _, _, loading := a.state.GetState()
			if loading {
				return
			}
			startIndex := (a.currentPage - 1) * a.pageSize
			globalIndex := startIndex + (row - 1)
			if globalIndex < len(a.totalSongs) {
				go a.playSongAtIndex(globalIndex)
			}
		}
	})

	a.setupKeyBindings()
	a.setupGlobalInputHandler()
}

func (a *App) setupKeyBindings() {
	km := a.keyBindings

	km.RegisterKeyBinding(
		KeyAction{name: "togglePlayPause", handler: a.handleSpaceKey},
		[]tcell.Key{},
		[]rune{' '},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "nextSong", handler: a.playNextSong},
		[]tcell.Key{tcell.KeyRight},
		[]rune{'n', 'N', 'l'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "prevSong", handler: a.playPreviousSong},
		[]tcell.Key{tcell.KeyLeft},
		[]rune{'p', 'P', 'h'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "nextPage", handler: a.nextPage},
		[]tcell.Key{tcell.KeyPgDn},
		[]rune{']', '>', 'J'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "prevPage", handler: a.previousPage},
		[]tcell.Key{tcell.KeyPgUp},
		[]rune{'[', '<', 'K'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "moveRowDown", handler: a.moveRowDown},
		[]tcell.Key{tcell.KeyDown},
		[]rune{'j'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "moveRowUp", handler: a.moveRowUp},
		[]tcell.Key{tcell.KeyUp},
		[]rune{'k'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "goStart", handler: a.goToFirstPage},
		[]tcell.Key{},
		[]rune{'G'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "goEnd", handler: a.goToLastPage},
		[]tcell.Key{},
		[]rune{'G'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "volumeUp", handler: a.volumeUp},
		[]tcell.Key{},
		[]rune{'+', '='},
	)
	km.RegisterKeyBinding(
		KeyAction{name: "volumeDown", handler: a.volumeDown},
		[]tcell.Key{},
		[]rune{'-', '_'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "search", handler: func() {
			a.tviewApp.SetFocus(a.searchInput)
		}},
		[]tcell.Key{},
		[]rune{'/'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "help", handler: a.showHelp},
		[]tcell.Key{},
		[]rune{'?'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "queue", handler: a.showQueue},
		[]tcell.Key{},
		[]rune{'q', 'Q'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "sort", handler: a.cycleSortMode},
		[]tcell.Key{},
		[]rune{'s'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "source", handler: a.cycleSongSource},
		[]tcell.Key{},
		[]rune{'S'},
	)
}

func (a *App) setupGlobalInputHandler() {
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.helpView != nil && a.helpView.IsActive() {
			if event.Key() == tcell.KeyEscape || event.Rune() == '?' {
				a.helpView.Close()
				return nil
			}
			return event
		}
		if a.queueView != nil && a.queueView.IsActive() {
			if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'Q' {
				a.queueView.Close()
				return nil
			}
			return event
		}

		if a.keyBindings.HandleKey(event) {
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			if a.isSearchMode {
				a.clearSearch()
				return nil
			}
			a.handleExit()
			return nil
		case tcell.KeyCtrlC:
			a.handleExit()
			return nil
		}

		return event
	})
}

func (a *App) handleSpaceKey() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("handleSpaceKey panic: %v", r)
			}
		}()

		currentSong, _, _, _ := a.state.GetState()
		if currentSong == nil {
			return
		}

		_, err := a.player.Pause()
		if err != nil {
			return
		}

		isPaused, err := a.player.IsPaused()
		if err != nil {
			return
		}

		newPlayingState := !isPaused
		a.state.SetPlaying(newPlayingState)

		if newPlayingState {
			a.updatePlayingDisplay(currentSong)
		} else {
			a.updatePausedDisplay(currentSong)
		}
	}()
}

func (a *App) handleExit() {
	if a.player != nil {
		a.player.Cleanup()
	}

	a.tviewApp.Stop()

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

func (a *App) performSearch(query string) {
	if !a.isSearchMode {
		a.originalSongs = make([]domain.Song, len(a.totalSongs))
		copy(a.originalSongs, a.totalSongs)
		a.isSearchMode = true
	}

	go func() {
		songs, err := a.library.SearchSongs(query, 100)
		if err != nil {
			a.tviewApp.QueueUpdateDraw(func() {
				a.statusBar.SetText(fmt.Sprintf("[red]Search failed: %v", err))
			})
			return
		}

		a.tviewApp.QueueUpdateDraw(func() {
			a.songsMu.Lock()
			a.totalSongs = songs
			a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
			if a.totalPages == 0 {
				a.totalPages = 1
			}
			a.currentPage = 1
			a.songsMu.Unlock()
			a.SortSongs()
			a.renderSongTable()
			a.updateStatusWithPageInfo()
			a.searchInput.SetFieldBackgroundColor(tcell.ColorDefault)
			a.tviewApp.SetFocus(a.songTable)
		})
	}()
}

func (a *App) clearSearch() {
	if a.isSearchMode {
		a.songsMu.Lock()
		a.totalSongs = a.originalSongs
		a.originalSongs = nil
		a.isSearchMode = false
		a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
		a.currentPage = 1
		a.songsMu.Unlock()
		a.renderSongTable()
		a.updateStatusWithPageInfo()
	}
	a.searchInput.SetText("")
	a.searchInput.SetFieldBackgroundColor(tcell.ColorDefault)
}

func (a *App) renderSongTable() {
	for i := a.songTable.GetRowCount() - 1; i > 0; i-- {
		a.songTable.RemoveRow(i)
	}
	a.setupTableHeaders()
	pageData := a.getCurrentPageData()
	startIndex := (a.currentPage - 1) * a.pageSize
	termWidth := a.getTerminalWidth()

	currentSong, _, isPlaying, _ := a.state.GetState()

	for i, song := range pageData {
		row := i + 1
		globalIndex := startIndex + i + 1

		isCurrentTrack := isPlaying && currentSong != nil && currentSong.ID == song.ID
		rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault)
		trackColor := tcell.NewHexColor(0xffb300)
		titleColor := tcell.ColorWhite
		if isCurrentTrack {
			trackColor = tcell.ColorLightGreen
			titleColor = tcell.ColorLightGreen
		}

		trackText := fmt.Sprintf("%d:", globalIndex)
		if isCurrentTrack {
			trackText = "▶"
		}

		trackCell := tview.NewTableCell(trackText).
			SetStyle(rowStyle.Foreground(trackColor)).
			SetAlign(tview.AlignRight)

		titleCell := tview.NewTableCell(song.Title).
			SetStyle(rowStyle.Foreground(titleColor)).
			SetExpansion(1)

		col := 0
		a.songTable.SetCell(row, col, trackCell)
		col++
		a.songTable.SetCell(row, col, titleCell)
		col++

		if termWidth >= 50 {
			durColor := tcell.ColorGray
			if isCurrentTrack {
				durColor = tcell.ColorLightGreen
			}
			durationCell := tview.NewTableCell(FormatDuration(song.Duration)).
				SetStyle(rowStyle.Foreground(durColor)).
				SetAlign(tview.AlignRight)
			a.songTable.SetCell(row, col, durationCell)
			col++
		}

		if termWidth >= 60 {
			artistWidth := termWidth / 6
			if artistWidth < 12 {
				artistWidth = 12
			}
			if artistWidth > 30 {
				artistWidth = 30
			}
			artistCell := tview.NewTableCell(song.Artist).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(artistWidth)
			a.songTable.SetCell(row, col, artistCell)
			col++
		}

		if termWidth >= 70 {
			albumWidth := termWidth / 6
			if albumWidth < 12 {
				albumWidth = 12
			}
			if albumWidth > 30 {
				albumWidth = 30
			}
			albumCell := tview.NewTableCell(song.Album).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(albumWidth)
			a.songTable.SetCell(row, col, albumCell)
		}
	}

	a.songTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewHexColor(0xffb300)).
		Foreground(tcell.ColorWhite))

	a.songTable.ScrollToBeginning()
}

func (a *App) updateProgressBar() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.tviewApp == nil {
				return
			}

			a.tickCount++

			currentSong, _, isPlaying, isLoading := a.state.GetState()
			if isLoading {
				continue
			}

			if !isPlaying {
				if currentSong != nil {
					a.updatePausedDisplay(currentSong)
				} else {
					a.updateIdleDisplay()
				}
				continue
			}

			if a.playingUpdateMu.TryLock() {
				go func() {
					defer a.playingUpdateMu.Unlock()
					a.updatePlayingDisplay(currentSong)
				}()
			}

		case <-a.ctx.Done():
			return
		}
	}
}

func (a *App) updateIdleDisplay() {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(CreateIdleBottomBar())
		}
	})
}

func (a *App) updatePausedDisplay(song *domain.Song) {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil && a.statusBar != nil {
			currentPos, totalDuration, err := a.player.GetProgress()
			if err != nil || totalDuration <= 0 {
				totalDuration = float64(song.Duration)
				currentPos = 0
			}

			currentTime := FormatDuration(int(currentPos))
			totalTime := FormatDuration(int(totalDuration))

			progress := currentPos / totalDuration
			if progress > 1 {
				progress = 1
			} else if progress < 0 {
				progress = 0
			}

			volumeText := "??"
			volumeVal := 0.0
			if vol, err := a.player.GetVolume(); err == nil {
				volumeText = fmt.Sprintf("%.0f%%", vol)
				volumeVal = vol
			}

			spinner := SpinnerChar(a.tickCount)
			volBar := CreateVolumeBar(volumeVal, 10)

			bottomBar := CreateBottomBar(progress, a.cfg.UI.ProgressBarWidth, currentTime, totalTime, volumeText,
				"[#ff9800]⏸ PAUSED", spinner)
			a.progressBar.SetText(bottomBar)

			pausedStatus := fmt.Sprintf("[#ff9800]⏸ PAUSED")
			a.statusBar.SetText(FormatSongInfo(*song, pausedStatus, spinner, volBar, a.leftPanelTextWidth(), a.serverConnected, CreatePausedExtras(*song, a.leftPanelTextWidth())))
		}
	})
}

func (a *App) updatePlayingDisplay(song *domain.Song) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("updatePlayingDisplay panic: %v", r)
		}
	}()

	if song == nil {
		return
	}

	currentPos, totalDuration, err := a.player.GetProgress()
	if err != nil || totalDuration <= 0 || currentPos < 0 {
		return
	}

	currentTime := FormatDuration(int(currentPos))
	totalTime := FormatDuration(int(totalDuration))

	progress := currentPos / totalDuration
	if progress > 1 {
		progress = 1
	} else if progress < 0 {
		progress = 0
	}

	volumeText := "??"
	volumeVal := 0.0
	if vol, err := a.player.GetVolume(); err == nil {
		volumeText = fmt.Sprintf("%.0f%%", vol)
		volumeVal = vol
	}

	spinner := SpinnerChar(a.tickCount)
	volBar := CreateVolumeBar(volumeVal, 10)

	bottomBar := CreateBottomBar(progress, a.cfg.UI.ProgressBarWidth, currentTime, totalTime, volumeText,
		"[#ffb300]▶ PLAYING", spinner)
	playingStatus := fmt.Sprintf("[#ffb300]▶ PLAYING")

	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(bottomBar)
		}
		if a.statusBar != nil {
			a.statusBar.SetText(FormatSongInfo(*song, playingStatus, spinner, volBar, a.leftPanelTextWidth(), a.serverConnected, CreatePlayingExtras(*song, a.leftPanelTextWidth())))
		}
	})
}

func (a *App) showHelp() {
	if a.helpView == nil {
		return
	}

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(a.helpView.GetContainer(), 60, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	a.tviewApp.SetRoot(modal, true)
	a.helpView.Show()
}

func (a *App) showQueue() {
	if a.queueView == nil {
		return
	}

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(a.queueView.GetContainer(), 80, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	a.tviewApp.SetRoot(modal, true)
	a.queueView.Show()
}
