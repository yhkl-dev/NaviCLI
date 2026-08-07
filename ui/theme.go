package ui

import "github.com/gdamore/tcell/v2"

// Theme: modern calm dark. Near-black canvas, soft gray text, one cool teal
// accent reserved strictly for live state: the playing track, the progress
// fill, the current time, and the connected dot. Panels separate by
// whitespace and typographic weight, not borders. State colors (amber for
// paused, red for error or disconnected) appear only as rare semantic
// signals, never as decoration.
//
// 60-30-10: 60% neutral canvas and primary text, 30% secondary/tertiary text
// and separators, 10% the teal accent on whatever is alive right now.
const (
	tagFg          = "#c9d1d9" // primary text, song title
	tagFgDim       = "#6e7681" // secondary text, column headers, total time
	tagFgFaint     = "#484f58" // tertiary text, hints, "vol" label
	tagAccent      = "#39c5cf" // teal: playing, progress fill, current time, connected
	tagAccentEmpty = "#1f3a3d" // dim teal-tinted: empty progress/volume cells
	tagWarn        = "#e3b341" // muted amber: paused
	tagDanger      = "#f87171" // muted red: error, disconnected
)

// tcell colors for cell styles (table text, headers, selection, current track).
var (
	colFg        = tcell.NewHexColor(0xc9d1d9)
	colFgDim     = tcell.NewHexColor(0x6e7681)
	colFgFaint   = tcell.NewHexColor(0x484f58)
	colAccent    = tcell.NewHexColor(0x39c5cf)
	colSelection = tcell.NewHexColor(0x15393d)
)
