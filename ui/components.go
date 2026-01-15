package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yhkl-dev/NaviCLI/domain"
)

// createHomepage sets up the UI layout
func (a *App) createHomepage() {
	a.progressBar = tview.NewTextView().
		SetDynamicColors(true)
	a.progressBar.SetBorder(false)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true)
	a.statusBar.SetBorder(false)

	// 创建搜索输入框
	a.searchInput = tview.NewInputField().
		SetLabel("[yellow]Search: ").
		SetFieldWidth(0).
		SetPlaceholder("Type to search, ESC to clear, ENTER to filter...").
		SetFieldBackgroundColor(tcell.ColorBlack)
	a.searchInput.SetBorder(false)

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.songTable.SetBorder(false)

	// Initialize views
	a.helpView = NewHelpView(a)
	a.queueView = NewQueueView(a)

	a.setupTableHeaders()
	a.setupSearchInput()
	a.setupInputHandlers()

	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.statusBar, 0, 1, false)

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.searchInput, 1, 0, false).
		AddItem(a.songTable, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftPanel, 0, 1, false).
		AddItem(rightPanel, 0, 2, true)

	a.rootFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.progressBar, 3, 0, false)

	a.tviewApp.SetRoot(a.rootFlex, true)
}

// setupTableHeaders sets up the table header row
func (a *App) setupTableHeaders() {
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Attributes(tcell.AttrBold)

	for col := 0; col < 5; col++ {
		a.songTable.SetCell(0, col, tview.NewTableCell("").SetStyle(headerStyle))
	}
}

// setupSearchInput sets up the search input field handlers
func (a *App) setupSearchInput() {
	a.searchInput.SetChangedFunc(func(text string) {
		if text == "" && a.isSearchMode {
			// 清空搜索，恢复原始列表
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
		// 按下箭头键时切换到歌曲列表
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyTab {
			a.tviewApp.SetFocus(a.songTable)
			return nil
		}
		return event
	})
}

// setupInputHandlers sets up keyboard input handlers
func (a *App) setupInputHandlers() {
	a.songTable.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			_, _, _, loading := a.state.GetState()
			if loading {
				return
			}
			// Calculate global index from page and row
			startIndex := (a.currentPage - 1) * a.pageSize
			globalIndex := startIndex + (row - 1)
			if globalIndex < len(a.totalSongs) {
				go a.playSongAtIndex(globalIndex)
			}
		}
	})

	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle modal views first
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

		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case ' ':
				a.handleSpaceKey()
				return nil
			case 'n', 'N':
				a.playNextSong()
				return nil
			case 'p', 'P':
				a.playPreviousSong()
				return nil
			case '/':
				a.tviewApp.SetFocus(a.searchInput)
				return nil
			case '?':
				a.showHelp()
				return nil
			case 'q', 'Q':
				a.showQueue()
				return nil
			case ']', '>':
				a.nextPage()
				return nil
			case '[', '<':
				a.previousPage()
				return nil
			case '+', '=':
				a.volumeUp()
				return nil
			case '-', '_':
				a.volumeDown()
				return nil
			}
		}

		switch event.Key() {
		case tcell.KeyEsc:
			// 如果在搜索模式，ESC 先清空搜索
			if a.isSearchMode {
				a.clearSearch()
				return nil
			}
			// 否则退出程序
			a.handleExit()
			return nil
		case tcell.KeyCtrlC:
			a.handleExit()
			return nil
		case tcell.KeyRight:
			a.playNextSong()
			return nil
		case tcell.KeyLeft:
			a.playPreviousSong()
			return nil
		case tcell.KeyPgDn:
			a.nextPage()
			return nil
		case tcell.KeyPgUp:
			a.previousPage()
			return nil
		}
		return event
	})
}

// handleSpaceKey handles the space key press (play/pause toggle)
func (a *App) handleSpaceKey() {
	go func() {
		defer func() {
			if recover() != nil {
			}
		}()

		currentSong, currentIndex, _, _ := a.state.GetState()
		if currentSong == nil {
			return
		}

		// 切换暂停状态
		_, err := a.player.Pause()
		if err != nil {
			return
		}

		// 获取实际的播放状态
		isPaused, err := a.player.IsPaused()
		if err != nil {
			return
		}

		// 更新应用状态（注意：isPaused=true 表示暂停，isPlaying应该是false）
		newPlayingState := !isPaused
		a.state.SetPlaying(newPlayingState)

		// 立即更新显示，使用与 updateProgressBar 相同的逻辑
		if newPlayingState {
			a.updatePlayingDisplay(currentSong, currentIndex)
		} else {
			a.updatePausedDisplay(currentSong, currentIndex)
		}
	}()
}

// handleExit handles the exit signal
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

// performSearch executes search and updates the table
func (a *App) performSearch(query string) {
	if !a.isSearchMode {
		// 保存原始列表
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
			a.totalSongs = songs
			a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
			if a.totalPages == 0 {
				a.totalPages = 1
			}
			a.currentPage = 1
			a.renderSongTable()
			a.updateStatusWithPageInfo()
			a.searchInput.SetFieldBackgroundColor(tcell.ColorDarkGreen)
			// 搜索完成后将焦点切回列表，以便能用上下键选择
			a.tviewApp.SetFocus(a.songTable)
		})
	}()
}

// clearSearch clears search and restores original list
func (a *App) clearSearch() {
	if a.isSearchMode {
		a.totalSongs = a.originalSongs
		a.originalSongs = nil
		a.isSearchMode = false
		a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
		a.currentPage = 1
		a.renderSongTable()
		a.updateStatusWithPageInfo()
	}
	a.searchInput.SetText("")
	a.searchInput.SetFieldBackgroundColor(tcell.ColorBlack)
}

// renderSongTable renders the song table with current page data
func (a *App) renderSongTable() {
	for i := a.songTable.GetRowCount() - 1; i > 0; i-- {
		a.songTable.RemoveRow(i)
	}
	a.setupTableHeaders()
	pageData := a.getCurrentPageData()
	startIndex := (a.currentPage - 1) * a.pageSize
	termWidth := a.getTerminalWidth()

	for i, song := range pageData {
		row := i + 1
		globalIndex := startIndex + i + 1
		rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault)

		trackCell := tview.NewTableCell(fmt.Sprintf("%d:", globalIndex)).
			SetStyle(rowStyle.Foreground(tcell.ColorLightGreen)).
			SetAlign(tview.AlignRight)

		titleCell := tview.NewTableCell(song.Title).
			SetStyle(rowStyle.Foreground(tcell.ColorWhite)).
			SetExpansion(1)

		col := 0
		a.songTable.SetCell(row, col, trackCell)
		col++
		a.songTable.SetCell(row, col, titleCell)
		col++

		if termWidth >= 50 {
			durationCell := tview.NewTableCell(FormatDuration(song.Duration)).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetAlign(tview.AlignRight)
			a.songTable.SetCell(row, col, durationCell)
			col++
		}

		if termWidth >= 60 {
			artistCell := tview.NewTableCell(song.Artist).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(15)
			a.songTable.SetCell(row, col, artistCell)
			col++
		}

		if termWidth >= 90 {
			albumCell := tview.NewTableCell(song.Album).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(15)
			a.songTable.SetCell(row, col, albumCell)
		}
	}

	a.songTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))

	a.songTable.ScrollToBeginning()
}

// updateProgressBar continuously updates the progress bar display
func (a *App) updateProgressBar() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.tviewApp == nil {
				return
			}

			currentSong, currentIndex, isPlaying, isLoading := a.state.GetState()
			if isLoading {
				continue
			}

			if !isPlaying {
				if currentSong != nil {
					a.updatePausedDisplay(currentSong, currentIndex)
				} else {
					a.updateIdleDisplay()
				}
				continue
			}

			go a.updatePlayingDisplay(currentSong, currentIndex)

		case <-time.After(15 * time.Second):
			if a.tviewApp == nil {
				return
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// updateIdleDisplay updates the display for idle state
func (a *App) updateIdleDisplay() {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(CreateIdleDisplay())
		}
	})
}

// updatePausedDisplay updates the display for paused state
func (a *App) updatePausedDisplay(song *domain.Song, index int) {
	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil && a.statusBar != nil {
			// 获取当前播放进度
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
			if vol, err := a.player.GetVolume(); err == nil {
				volumeText = fmt.Sprintf("%.0f%%", vol)
			}

			pausedProgressText := fmt.Sprintf(`
[darkgray]%s/%s [darkgray][v-] [white]%s [darkgray][v+] [darkgray][paused]`, currentTime, totalTime, volumeText)
			a.progressBar.SetText(pausedProgressText)

			pausedStatus := fmt.Sprintf("[yellow]%s [darkgray](PAUSED)", song.Title)
			progressBar := CreateProgressBar(progress, a.cfg.UI.ProgressBarWidth)
			a.statusBar.SetText(FormatSongInfo(*song, index, pausedStatus, progressBar))
		}
	})
}

// updatePlayingDisplay updates the display for playing state
func (a *App) updatePlayingDisplay(song *domain.Song, index int) {
	defer func() {
		if recover() != nil {
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
	if vol, err := a.player.GetVolume(); err == nil {
		volumeText = fmt.Sprintf("%.0f%%", vol)
	}

	progressText := CreateProgressText(currentTime, totalTime, volumeText)
	progressBar := CreateProgressBar(progress, a.cfg.UI.ProgressBarWidth)
	playingStatus := fmt.Sprintf("[lightgreen]%s", song.Title)

	a.tviewApp.QueueUpdateDraw(func() {
		if a.progressBar != nil {
			a.progressBar.SetText(progressText)
		}
		if a.statusBar != nil {
			// 使用带封面的格式，如果有封面的话
			a.statusBar.SetText(FormatSongInfo(*song, index, playingStatus, progressBar))
		}
	})
}

// showHelp displays the help modal view
func (a *App) showHelp() {
	if a.helpView == nil {
		return
	}

	// Create modal container
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(a.helpView.GetContainer(), 60, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	// Add modal to root
	a.tviewApp.SetRoot(modal, true)
	a.helpView.Show()
}

// showQueue displays the queue modal view
func (a *App) showQueue() {
	if a.queueView == nil {
		return
	}

	// Create modal container
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(a.queueView.GetContainer(), 80, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	// Add modal to root
	a.tviewApp.SetRoot(modal, true)
	a.queueView.Show()
}
