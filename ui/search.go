package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yhkl-dev/NaviCLI/domain"
)

// SearchView represents the search interface
type SearchView struct {
	app         *App
	container   *tview.Flex
	inputField  *tview.InputField
	resultTable *tview.Table
	results     []domain.Song
	isActive    bool
}

// NewSearchView creates a new search view
func NewSearchView(app *App) *SearchView {
	sv := &SearchView{
		app:     app,
		results: make([]domain.Song, 0),
	}

	sv.inputField = tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault)

	sv.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			sv.performSearch()
		} else if key == tcell.KeyEscape {
			sv.Close()
		}
	})

	sv.resultTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	sv.resultTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(sv.results) {
			song := sv.results[row-1]
			sv.playSong(song)
			sv.Close()
		}
	})

	sv.resultTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			sv.Close()
			return nil
		}

		// 获取当前选中的行
		row, _ := sv.resultTable.GetSelection()
		if row > 0 && row-1 < len(sv.results) {
			song := sv.results[row-1]

			// Enter - 立即播放
			if event.Key() == tcell.KeyEnter {
				sv.playSong(song)
				sv.Close()
				return nil
			}

			// N - 添加到下一首播放
			if event.Rune() == 'n' || event.Rune() == 'N' {
				sv.playNext(song)
				sv.Close()
				return nil
			}
		}

		return event
	})

	// Setup header
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Attributes(tcell.AttrBold)
	sv.resultTable.SetCell(0, 0, tview.NewTableCell("#").SetStyle(headerStyle))
	sv.resultTable.SetCell(0, 1, tview.NewTableCell("Title").SetStyle(headerStyle))
	sv.resultTable.SetCell(0, 2, tview.NewTableCell("Artist").SetStyle(headerStyle))
	sv.resultTable.SetCell(0, 3, tview.NewTableCell("Album").SetStyle(headerStyle))
	sv.resultTable.SetCell(0, 4, tview.NewTableCell("Duration").SetStyle(headerStyle))

	sv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(sv.inputField, 1, 0, true).
		AddItem(sv.resultTable, 0, 1, false)

	sv.container.SetBorder(true).
		SetTitle(" Search [ENTER: Play | N: Play Next | ESC: Close] ").
		SetBorderColor(tcell.ColorGreen)

	// Capture ESC at container level to ensure it works regardless of focus
	sv.container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			sv.Close()
			return nil
		}
		return event
	})

	return sv
}

// Show displays the search view
func (sv *SearchView) Show() {
	sv.isActive = true
	sv.app.tviewApp.SetFocus(sv.inputField)
}

// Close hides the search view
func (sv *SearchView) Close() {
	sv.isActive = false
	sv.inputField.SetText("")
	sv.results = make([]domain.Song, 0)
	sv.clearResults()
	sv.app.tviewApp.SetRoot(sv.app.rootFlex, true)
	sv.app.tviewApp.SetFocus(sv.app.songTable)
}

// IsActive returns whether the search view is active
func (sv *SearchView) IsActive() bool {
	return sv.isActive
}

// GetContainer returns the search view container
func (sv *SearchView) GetContainer() *tview.Flex {
	return sv.container
}

// performSearch executes the search query
func (sv *SearchView) performSearch() {
	query := sv.inputField.GetText()
	if query == "" {
		return
	}

	go func() {
		songs, err := sv.app.library.SearchSongs(query, 50)
		if err != nil {
			sv.app.tviewApp.QueueUpdateDraw(func() {
				sv.showError(fmt.Sprintf("Search failed: %v", err))
			})
			return
		}

		sv.results = songs
		sv.app.tviewApp.QueueUpdateDraw(func() {
			sv.displayResults()
			sv.app.tviewApp.SetFocus(sv.resultTable)
		})
	}()
}

// displayResults renders the search results in the table
func (sv *SearchView) displayResults() {
	// Clear previous results
	sv.clearResults()

	if len(sv.results) == 0 {
		sv.resultTable.SetCell(1, 0, tview.NewTableCell("No results found").
			SetAlign(tview.AlignCenter).
			SetExpansion(5))
		return
	}

	rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	for i, song := range sv.results {
		row := i + 1

		sv.resultTable.SetCell(row, 0,
			tview.NewTableCell(fmt.Sprintf("%d", i+1)).
				SetStyle(rowStyle.Foreground(tcell.ColorLightGreen)).
				SetAlign(tview.AlignRight))

		sv.resultTable.SetCell(row, 1,
			tview.NewTableCell(song.Title).
				SetStyle(rowStyle).
				SetExpansion(2))

		sv.resultTable.SetCell(row, 2,
			tview.NewTableCell(song.Artist).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(20))

		sv.resultTable.SetCell(row, 3,
			tview.NewTableCell(song.Album).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(20))

		sv.resultTable.SetCell(row, 4,
			tview.NewTableCell(FormatDuration(song.Duration)).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetAlign(tview.AlignRight))
	}

	sv.resultTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))
}

// clearResults clears the result table
func (sv *SearchView) clearResults() {
	for i := sv.resultTable.GetRowCount() - 1; i > 0; i-- {
		sv.resultTable.RemoveRow(i)
	}
}

// showError displays an error message
func (sv *SearchView) showError(message string) {
	sv.clearResults()
	sv.resultTable.SetCell(1, 0, tview.NewTableCell(message).
		SetTextColor(tcell.ColorRed).
		SetAlign(tview.AlignCenter).
		SetExpansion(5))
}

// playSong plays the selected song from search results
func (sv *SearchView) playSong(song domain.Song) {
	// Find song index in total songs, or add to end
	index := -1
	for i, s := range sv.app.totalSongs {
		if s.ID == song.ID {
			index = i
			break
		}
	}

	if index == -1 {
		// Song not in current list, add it
		sv.app.totalSongs = append(sv.app.totalSongs, song)
		index = len(sv.app.totalSongs) - 1
	}

	sv.app.playSongAtIndex(index)
}

// playNext adds the selected song to play next (after current song)
func (sv *SearchView) playNext(song domain.Song) {
	// Check if song already exists in list
	existingIndex := -1
	for i, s := range sv.app.totalSongs {
		if s.ID == song.ID {
			existingIndex = i
			break
		}
	}

	// Get current playing index
	_, currentIndex, _, _ := sv.app.state.GetState()

	// If song already exists, remove it first
	if existingIndex != -1 {
		sv.app.totalSongs = append(sv.app.totalSongs[:existingIndex], sv.app.totalSongs[existingIndex+1:]...)
		// Adjust currentIndex if necessary
		if existingIndex <= currentIndex {
			currentIndex--
		}
	}

	// Insert after current song
	insertPos := currentIndex + 1
	if insertPos > len(sv.app.totalSongs) {
		insertPos = len(sv.app.totalSongs)
	}

	// Insert song at position
	sv.app.totalSongs = append(sv.app.totalSongs[:insertPos], append([]domain.Song{song}, sv.app.totalSongs[insertPos:]...)...)

	// Update total pages
	sv.app.totalPages = (len(sv.app.totalSongs) + sv.app.pageSize - 1) / sv.app.pageSize

	// Refresh the display
	sv.app.tviewApp.QueueUpdateDraw(func() {
		sv.app.renderSongTable()
		sv.app.updateStatusWithPageInfo()
	})
}
