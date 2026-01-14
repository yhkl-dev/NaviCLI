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

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	a.songTable.SetBorder(false)

	// Initialize views
	a.searchView = NewSearchView(a)
	a.helpView = NewHelpView(a)
	a.queueView = NewQueueView(a)

	a.setupTableHeaders()
	a.setupInputHandlers()

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

	a.tviewApp.SetRoot(a.rootFlex, true)
}

// setupTableHeaders sets up the table header row
func (a *App) setupTableHeaders() {
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Attributes(tcell.AttrBold)

	for col := 0; col < 5; col++ {
		a.songTable.SetCell(0, col, tview.NewTableCell("").SetStyle(headerStyle))
	}
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
		if a.searchView != nil && a.searchView.IsActive() {
			return event // Let search view handle its own input
		}
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
				a.showSearch()
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
		case tcell.KeyEsc, tcell.KeyCtrlC:
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

		currentSong, currentIndex, isPlaying, _ := a.state.GetState()
		if currentSong == nil {
			return
		}

		a.player.Pause()
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

		info := FormatSongInfo(*currentSong, currentIndex, status, progressBar)
		a.updateStatus(info)
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
			volumeText := "??"
			if vol, err := a.player.GetVolume(); err == nil {
				volumeText = fmt.Sprintf("%.0f%%", vol)
			}

			pausedDisplay := fmt.Sprintf(`
[darkgray]00:00:00 [darkgray][v-] [white]%s [darkgray][v+] [darkgray][random]`, volumeText)
			a.progressBar.SetText(pausedDisplay)

			pausedStatus := fmt.Sprintf("[yellow]%s [darkgray](PAUSED)", song.Title)
			progressBar := "[darkgray]▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░ 0%"
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
			if a.currentCover != "" {
				a.statusBar.SetText(FormatSongInfoWithCover(*song, index, playingStatus, progressBar, a.currentCover))
			} else {
				a.statusBar.SetText(FormatSongInfo(*song, index, playingStatus, progressBar))
			}
		}
	})
}

// showSearch displays the search modal view
func (a *App) showSearch() {
	if a.searchView == nil {
		return
	}

	// Create modal container
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(a.searchView.GetContainer(), 80, 0, true).
			AddItem(nil, 0, 1, false), 20, 0, true).
		AddItem(nil, 0, 1, false)

	// Add modal to root
	a.tviewApp.SetRoot(modal, true)
	a.searchView.Show()
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
