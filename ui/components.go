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
		SetLabel("[" + tagFgFaint + "]/ ").
		SetFieldWidth(0).
		SetPlaceholder("search").
		SetFieldBackgroundColor(tcell.ColorDefault)
	a.searchInput.SetBorder(false)

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(0, 0)
	a.songTable.SetBorder(false)

	a.helpView = NewHelpView(a)

	a.setupSearchInput()
	a.setupInputHandlers()

	a.leftTitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[" + tagFg + "]Now playing")

	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.leftTitleBar, 1, 0, false).
		AddItem(a.statusBar, 0, 1, false)

	a.rightTitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[" + tagFg + "]Library")

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.rightTitleBar, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(a.searchInput, 1, 0, false).
		AddItem(nil, 1, 0, false).
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

	// Render the title bars with sort mode + connection dot. Without this the
	// headers stay bare until the first sort cycle or connection change.
	a.updateSortTitle()
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
		if row >= 0 {
			_, _, _, loading := a.state.GetState()
			if loading {
				return
			}
			startIndex := (a.currentPage - 1) * a.pageSize
			globalIndex := startIndex + row/2
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
		KeyAction{name: "seekBack", handler: a.seekBackward},
		[]tcell.Key{},
		[]rune{','},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "seekForward", handler: a.seekForward},
		[]tcell.Key{},
		[]rune{'.'},
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
		KeyAction{name: "sort", handler: a.cycleSortMode},
		[]tcell.Key{},
		[]rune{'s'},
	)

	km.RegisterKeyBinding(
		KeyAction{name: "repeat", handler: a.cycleRepeat},
		[]tcell.Key{},
		[]rune{'r'},
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
				a.statusBar.SetText(fmt.Sprintf("["+tagDanger+"]Search failed: %v", err))
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
	for i := a.songTable.GetRowCount() - 1; i >= 0; i-- {
		a.songTable.RemoveRow(i)
	}
	termWidth := a.getTerminalWidth()
	pageData := a.getCurrentPageData()

	currentSong, _, isPlaying, _ := a.state.GetState()

	// Cap the title/artist column so a long title cannot push the Time column
	// off-screen. The table fills ~3/4 of the terminal; reserve the indicator
	// (1) and Time (6) columns plus padding.
	titleMax := termWidth*3/4 - 8
	if titleMax < 10 {
		titleMax = 10
	}

	for i, song := range pageData {
		titleRow := 2 * i
		metaRow := 2*i + 1
		isCurrentTrack := isPlaying && currentSong != nil && currentSong.ID == song.ID

		rowStyle := tcell.StyleDefault.Foreground(colFg).Background(tcell.ColorDefault)
		indicator := " "
		indicatorColor := colFgDim
		titleColor := colFg
		if isCurrentTrack {
			indicator = "▶"
			indicatorColor = colAccent
			titleColor = colAccent
		}

		// Title row: marker, title, time.
		a.songTable.SetCell(titleRow, 0, tview.NewTableCell(indicator).
			SetStyle(rowStyle.Foreground(indicatorColor)))
		a.songTable.SetCell(titleRow, 1, tview.NewTableCell(song.Title).
			SetStyle(rowStyle.Foreground(titleColor).Attributes(tcell.AttrBold)).
			SetExpansion(1).
			SetMaxWidth(titleMax))
		if termWidth >= 40 {
			durColor := colFgDim
			if isCurrentTrack {
				durColor = colAccent
			}
			a.songTable.SetCell(titleRow, 2, tview.NewTableCell(FormatDuration(song.Duration)).
				SetStyle(rowStyle.Foreground(durColor)).
				SetAlign(tview.AlignRight))
		}

		// Meta row: artist · album, faint and indented under the title.
		meta := song.Artist
		if song.Album != "" {
			if meta != "" {
				meta += "  ·  "
			}
			meta += song.Album
		}
		a.songTable.SetCell(metaRow, 0, tview.NewTableCell(" ").SetStyle(rowStyle))
		a.songTable.SetCell(metaRow, 1, tview.NewTableCell("  "+meta).
			SetStyle(rowStyle.Foreground(colFgFaint)).
			SetExpansion(1).
			SetMaxWidth(titleMax))
	}

	a.songTable.SetSelectedStyle(tcell.StyleDefault.
		Background(colSelection).
		Foreground(colFg))

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

			a.tickCount.Add(1)

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
			currentPos, _, err := a.player.GetProgress()
			if err != nil || currentPos < 0 {
				currentPos = 0
			}

			totalDuration := float64(song.Duration)
			currentTime := FormatDuration(int(currentPos))
			totalTime := FormatDuration(song.Duration)

			progress := 0.0
			if totalDuration > 0 {
				progress = currentPos / totalDuration
			}
			if progress > 1 {
				progress = 1
			}

			volumeVal := 0.0
			if vol, err := a.player.GetVolume(); err == nil {
				volumeVal = vol
			}

			spinner := SpinnerChar(int(a.tickCount.Load()))
			bottomBar := CreateBottomBar(progress, a.cfg.UI.ProgressBarWidth, currentTime, totalTime, volumeVal,
				"["+tagWarn+"]⏸ PAUSED", spinner)
			a.progressBar.SetText(bottomBar)
			a.statusBar.SetText(a.nowPlaying(*song, CreateTrackExtras(*song)))
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

	currentPos, _, err := a.player.GetProgress()
	if err != nil || currentPos < 0 {
		return
	}

	totalDuration := float64(song.Duration)
	if totalDuration <= 0 {
		return
	}

	currentTime := FormatDuration(int(currentPos))
	totalTime := FormatDuration(song.Duration)

	progress := currentPos / totalDuration
	if progress > 1 {
		progress = 1
	}

	volumeVal := 0.0
	if vol, err := a.player.GetVolume(); err == nil {
		volumeVal = vol
	}

	spinner := SpinnerChar(int(a.tickCount.Load()))
	bottomBar := CreateBottomBar(progress, a.cfg.UI.ProgressBarWidth, currentTime, totalTime, volumeVal,
		"["+tagAccent+"]▶ PLAYING", spinner)

	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(bottomBar)
		}
		if a.statusBar != nil {
			a.statusBar.SetText(a.nowPlaying(*song, CreateTrackExtras(*song)))
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
