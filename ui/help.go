package ui

import (
	"strings"

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

	// [$h] tokens bind to the primary text color so the help text never
	// hardcodes a hex value; the prefix match covers both [$h] and [$h::b]
	// (the bold title).
	helpText := strings.ReplaceAll(`[$h::b]Keyboard shortcuts[-:-:-]

[$h]Playback controls[-]
  [white]Space[-]       Play/Pause current song
  [white]Enter[-]       Play selected song
  [white]n / N / l[-]   Next song
  [white]p / P / h[-]   Previous song
  [white]-> / ←[-]       Next/Previous song (arrow keys)
  [white]+ / =[-]       Volume up (+5%)
  [white]- / _[-]       Volume down (-5%)
  [white], / .[-]       Seek -10s / +10s
  [white]r[-]           Repeat: off / all / one

[$h]Navigation (Vim-style)[-]
  [white]j / ↓[-]       Move down in list
  [white]k / ↑[-]       Move up in list
  [white]J / PgDn[-]    Next page
  [white]K / PgUp[-]    Previous page
  [white]> / ][-]       Next page (alternative)
  [white]< / [[-]       Previous page (alternative)
  [white]gg[-]          Go to first page
  [white]G[-]           Go to last page

[$h]Search & info[-]
  [white]/[-]           Open search
  [white]s[-]           Sort: Random / Title / Artist / Album
  [white]?[-]           Show this help panel

[$h]General[-]
  [white]ESC[-]         Close modal / Exit program
  [white]Ctrl+C[-]      Exit program

[$h]Press ESC or ? to close this help panel[-]
`, "[$h", "["+tagFg)

	hv.textView.SetText(helpText)

	hv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(hv.textView, 0, 1, true)

	hv.container.SetBorder(true).
		SetTitle(" Help (ESC to close) ").
		SetBorderColor(colFgDim).
		SetTitleColor(colAccent)

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
