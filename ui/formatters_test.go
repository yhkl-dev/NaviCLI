package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/yhkl-dev/NaviCLI/domain"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "0:00"},
		{5, "0:05"},
		{65, "1:05"},
		{125, "2:05"},
		{3600, "1:00:00"},
		{7384, "2:03:04"},
	}
	for _, tc := range tests {
		if got := FormatDuration(tc.seconds); got != tc.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestSpinnerChar(t *testing.T) {
	frames := []string{"◴", "◷", "◶", "◵"}
	for i, want := range frames {
		if got := SpinnerChar(i); got != want {
			t.Errorf("SpinnerChar(%d) = %q, want %q", i, got, want)
		}
	}
	if got := SpinnerChar(len(frames)); got != frames[0] {
		t.Errorf("SpinnerChar(%d) = %q, want %q (wrap)", len(frames), got, frames[0])
	}
}

// colorCount reports how many tview tags for the given hex color appear in s.
func colorCount(s, hexColor string) int {
	return strings.Count(s, "["+hexColor+"]")
}

func TestBlockBar(t *testing.T) {
	// Empty fraction: every cell is the dim empty tone, none accent.
	bar := blockBar(0, 4)
	if c := colorCount(bar, tagAccent); c != 0 {
		t.Errorf("blockBar(0,4): want 0 accent cells, got %d", c)
	}
	if c := colorCount(bar, tagAccentEmpty); c != 4 {
		t.Errorf("blockBar(0,4): want 4 empty cells, got %d", c)
	}

	// Full fraction: every cell accent.
	bar = blockBar(1, 4)
	if c := colorCount(bar, tagAccent); c != 4 {
		t.Errorf("blockBar(1,4): want 4 accent cells, got %d", c)
	}
	if c := colorCount(bar, tagAccentEmpty); c != 0 {
		t.Errorf("blockBar(1,4): want 0 empty cells, got %d", c)
	}

	// Half on 8 cells: 4 filled, 4 empty.
	bar = blockBar(0.5, 8)
	if c := colorCount(bar, tagAccent); c != 4 {
		t.Errorf("blockBar(0.5,8): want 4 accent cells, got %d", c)
	}
	if c := colorCount(bar, tagAccentEmpty); c != 4 {
		t.Errorf("blockBar(0.5,8): want 4 empty cells, got %d", c)
	}

	// Out-of-range fractions clamp, never panic or NaN.
	for _, frac := range []float64{-1, -0.01, 1.5, 2, 100} {
		bar = blockBar(frac, 4)
		if c := colorCount(bar, tagAccent) + colorCount(bar, tagAccentEmpty); c != 4 {
			t.Errorf("blockBar(%v,4): want 4 total cells, got %d", frac, c)
		}
	}

	// NaN must not crash; int(NaN) yields 0, so all cells empty.
	bar = blockBar(math.NaN(), 4)
	if c := colorCount(bar, tagAccentEmpty); c != 4 {
		t.Errorf("blockBar(NaN,4): want 4 empty cells, got %d", c)
	}

	// Width below the floor is clamped to 4 cells.
	bar = blockBar(0, 2)
	if c := colorCount(bar, tagAccentEmpty); c != 4 {
		t.Errorf("blockBar(0,2): want 4 empty cells (width clamped), got %d", c)
	}
}

func TestCreateVolumeBar(t *testing.T) {
	for _, tc := range []struct {
		volume float64
		want   string
	}{
		{0, "0%"},
		{50, "50%"},
		{75, "75%"},
		{100, "100%"},
	} {
		got := CreateVolumeBar(tc.volume, 10)
		if !strings.Contains(got, "vol") {
			t.Errorf("CreateVolumeBar(%v): missing label", tc.volume)
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("CreateVolumeBar(%v) = %q, want it to contain %q", tc.volume, got, tc.want)
		}
	}
	// Over-range volume clamps to a full accent bar, no empty cells.
	bar := CreateVolumeBar(150, 10)
	if c := colorCount(bar, tagAccentEmpty); c != 0 {
		t.Errorf("CreateVolumeBar(150): want 0 empty cells, got %d", c)
	}
}

func TestCreateProgressBar(t *testing.T) {
	// Zero progress is all empty; full is all accent.
	if c := colorCount(CreateProgressBar(0, 10), tagAccentEmpty); c != 10 {
		t.Errorf("CreateProgressBar(0,10): want 10 empty cells, got %d", c)
	}
	if c := colorCount(CreateProgressBar(1, 10), tagAccent); c != 10 {
		t.Errorf("CreateProgressBar(1,10): want 10 accent cells, got %d", c)
	}
	// Width floor is 10.
	if c := colorCount(CreateProgressBar(0, 4), tagAccentEmpty); c != 10 {
		t.Errorf("CreateProgressBar(0,4): want 10 empty cells (width floored), got %d", c)
	}
}

func TestCreateBottomBar(t *testing.T) {
	got := CreateBottomBar(0.5, 20, "1:30", "3:00", 75, "["+tagAccent+"]▶ PLAYING", "◴")
	for _, s := range []string{"1:30", "3:00", "vol", "75%", "▶ PLAYING", "◴"} {
		if !strings.Contains(got, s) {
			t.Errorf("CreateBottomBar: missing %q in %q", s, got)
		}
	}
	// Progress is shown by the bar fill, not a duplicate percentage number.
	if strings.Contains(got, "50.0%") {
		t.Errorf("CreateBottomBar should not render a progress percentage: %q", got)
	}
}

func TestCreateIdleBottomBar(t *testing.T) {
	got := CreateIdleBottomBar()
	for _, s := range []string{"Ready", "Select a song", "ENTER"} {
		if !strings.Contains(got, s) {
			t.Errorf("CreateIdleBottomBar: missing %q", s)
		}
	}
}

func TestCreateWelcomeMessage(t *testing.T) {
	msg := CreateWelcomeMessage(42)
	for _, s := range []string{"NaviCLI", "42 songs loaded", "search", "navigate"} {
		if !strings.Contains(msg, s) {
			t.Errorf("CreateWelcomeMessage(42): missing %q", s)
		}
	}
	// Connection status moved to the title bar; it must not appear here.
	if strings.Contains(msg, "Navidrome connected") || strings.Contains(msg, "Navidrome disconnected") {
		t.Error("CreateWelcomeMessage should not render connection status (moved to title bar)")
	}
}

func TestFormatSongInfo(t *testing.T) {
	track := domain.Song{
		Title:    "Test Song",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Duration: 195,
		Track:    3,
	}
	got := FormatSongInfo(track, 40, "", CreateTrackExtras(track))
	for _, s := range []string{"Test Song", "Test Artist", "Test Album"} {
		if !strings.Contains(got, s) {
			t.Errorf("FormatSongInfo: missing %q in %q", s, got)
		}
	}
	// Track number and duration moved to the list and bottom bar, not the panel.
	if strings.Contains(got, "Track #3") || strings.Contains(got, "3:15") {
		t.Errorf("FormatSongInfo should not render track number or duration: %q", got)
	}
	// Connection status moved to the title bar.
	if strings.Contains(got, "Navidrome connected") {
		t.Error("FormatSongInfo should not render connection status")
	}
	// Cover block renders above the title when provided.
	withCover := FormatSongInfo(track, 40, "  COVERIMG\n", "")
	if !strings.Contains(withCover, "COVERIMG") {
		t.Errorf("FormatSongInfo: cover block not rendered in %q", withCover)
	}
	// Extra block is appended when provided.
	withExtra := FormatSongInfo(track, 40, "", "  SPEC LINE")
	if !strings.Contains(withExtra, "SPEC LINE") {
		t.Errorf("FormatSongInfo: extra block not rendered in %q", withExtra)
	}
}

func TestCreateTrackExtras(t *testing.T) {
	full := domain.Song{
		Suffix:       "flac",
		SampleRate:   44100,
		BitRate:      1411,
		ChannelCount: 2,
		Size:         50000000,
		PlayCount:    3,
	}
	got := CreateTrackExtras(full)
	for _, s := range []string{"FLAC", "44.1kHz", "1411kbps", "Stereo", "47.7 MB", "Plays", tagFgDim} {
		if !strings.Contains(got, s) {
			t.Errorf("full track: missing %q in %q", s, got)
		}
	}

	mono := CreateTrackExtras(domain.Song{Suffix: "mp3", ChannelCount: 1})
	if !strings.Contains(mono, "Mono") {
		t.Errorf("mono: missing %q in %q", "Mono", mono)
	}

	multi := CreateTrackExtras(domain.Song{Suffix: "mp3", ChannelCount: 6})
	if !strings.Contains(multi, "6ch") {
		t.Errorf("multi: missing %q in %q", "6ch", multi)
	}

	// Empty/zero fields are omitted, unknown format renders "?".
	sparse := CreateTrackExtras(domain.Song{})
	for _, s := range []string{"Mono", "?"} {
		if !strings.Contains(sparse, s) {
			t.Errorf("sparse: missing %q in %q", s, sparse)
		}
	}
	for _, s := range []string{"kHz", "kbps", "MB", "plays"} {
		if strings.Contains(sparse, s) {
			t.Errorf("sparse: should not contain %q (zero fields omitted)", s)
		}
	}
}
