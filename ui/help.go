package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpView struct {
	app       *App
	container *tview.Flex
	textView  *tview.TextView
	isActive  bool
}

func NewHelpView(app *App) *HelpView {
	hv := &HelpView{
		app: app,
	}

	hv.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)

	helpText := `[yellow::b]Keyboard Shortcuts[-:-:-]

[lightgreen]Playback Controls:[-]
  [white]Space[-]       Play/Pause current song
  [white]Enter[-]       Play selected song
  [white]n / N / l[-]   Next song
  [white]p / P / h[-]   Previous song
  [white]→ / ←[-]       Next/Previous song (arrow keys)
  [white]+ / =[-]       Volume up (+5%)
  [white]- / _[-]       Volume down (-5%)

[lightgreen]Navigation (Vim-style):[-]
  [white]j / ↓[-]       Move down in list
  [white]k / ↑[-]       Move up in list
  [white]J / PgDn[-]    Next page
  [white]K / PgUp[-]    Previous page
  [white]> / ][-]       Next page (alternative)
  [white]< / [[-]       Previous page (alternative)
  [white]gg[-]          Go to first page
  [white]G[-]           Go to last page

[lightgreen]Search & Info:[-]
  [white]/[-]           Open search
  [white]?[-]           Show this help panel
  [white]q / Q[-]       Show playback queue

[lightgreen]General:[-]
  [white]ESC[-]         Close modal / Exit program
  [white]Ctrl+C[-]      Exit program

[yellow]Press ESC or ? to close this help panel[-]
`

	hv.textView.SetText(helpText)

	hv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(hv.textView, 0, 1, true)

	hv.container.SetBorder(true).
		SetTitle(" Help (ESC to close) ").
		SetBorderColor(tcell.ColorYellow)

	return hv
}

func (hv *HelpView) Show() {
	hv.isActive = true
	hv.app.tviewApp.SetFocus(hv.textView)
}

func (hv *HelpView) Close() {
	hv.isActive = false
	hv.app.tviewApp.SetRoot(hv.app.rootFlex, true)
	hv.app.tviewApp.SetFocus(hv.app.songTable)
}

func (hv *HelpView) IsActive() bool {
	return hv.isActive
}

func (hv *HelpView) GetContainer() *tview.Flex {
	return hv.container
}
