package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// QueueView represents the playback queue interface
type QueueView struct {
	app       *App
	container *tview.Flex
	table     *tview.Table
	isActive  bool
}

// NewQueueView creates a new queue view
func NewQueueView(app *App) *QueueView {
	qv := &QueueView{
		app: app,
	}

	qv.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// Setup header
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Attributes(tcell.AttrBold)
	qv.table.SetCell(0, 0, tview.NewTableCell("#").SetStyle(headerStyle))
	qv.table.SetCell(0, 1, tview.NewTableCell("Title").SetStyle(headerStyle))
	qv.table.SetCell(0, 2, tview.NewTableCell("Artist").SetStyle(headerStyle))
	qv.table.SetCell(0, 3, tview.NewTableCell("Duration").SetStyle(headerStyle))

	qv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(qv.table, 0, 1, true)

	qv.container.SetBorder(true).
		SetTitle(" Playback Queue (ESC/q to close) ").
		SetBorderColor(tcell.NewHexColor(0x00bcd4))

	return qv
}

// Show displays the queue view
func (qv *QueueView) Show() {
	qv.isActive = true
	qv.refreshQueue()
	qv.app.tviewApp.SetFocus(qv.table)
}

// Close hides the queue view
func (qv *QueueView) Close() {
	qv.isActive = false
	qv.app.tviewApp.SetRoot(qv.app.rootFlex, true)
	qv.app.tviewApp.SetFocus(qv.app.songTable)
}

// IsActive returns whether the queue view is active
func (qv *QueueView) IsActive() bool {
	return qv.isActive
}

// GetContainer returns the queue view container
func (qv *QueueView) GetContainer() *tview.Flex {
	return qv.container
}

// refreshQueue updates the queue display with current items
func (qv *QueueView) refreshQueue() {
	// Clear existing rows
	for i := qv.table.GetRowCount() - 1; i > 0; i-- {
		qv.table.RemoveRow(i)
	}

	queue := qv.app.player.GetQueue()

	if len(queue) == 0 {
		qv.table.SetCell(1, 0, tview.NewTableCell("Queue is empty").
			SetAlign(tview.AlignCenter).
			SetExpansion(4).
			SetTextColor(tcell.ColorGray))
		return
	}

	rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	for i, item := range queue {
		row := i + 1

		qv.table.SetCell(row, 0,
			tview.NewTableCell(fmt.Sprintf("%d", i+1)).
				SetStyle(rowStyle.Foreground(tcell.ColorLightGreen)).
				SetAlign(tview.AlignRight))

		qv.table.SetCell(row, 1,
			tview.NewTableCell(item.Title).
				SetStyle(rowStyle).
				SetExpansion(2))

		qv.table.SetCell(row, 2,
			tview.NewTableCell(item.Artist).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetMaxWidth(20))

		qv.table.SetCell(row, 3,
			tview.NewTableCell(FormatDuration(item.Duration)).
				SetStyle(rowStyle.Foreground(tcell.ColorGray)).
				SetAlign(tview.AlignRight))
	}

	qv.table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))
}
