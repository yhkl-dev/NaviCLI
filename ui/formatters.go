package ui

import (
	"fmt"
	"strings"

	"github.com/yhkl-dev/NaviCLI/domain"
)

var spinnerFrames = []string{"◴", "◷", "◶", "◵"}

func SpinnerChar(tick int) string {
	return spinnerFrames[tick%len(spinnerFrames)]
}

func FormatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// blockBar renders a 0..1 fraction as a cell-resolution bar of the given
// width. Filled cells use the teal accent, empty cells a dim teal-tinted
// tone. One visual language for progress and volume. Coarser than braille
// but cleaner at small sizes: it reads as a solid bar, not dotted noise.
func blockBar(fraction float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int(fraction * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			fmt.Fprintf(&bar, "[%s]▰", tagAccent)
		} else {
			fmt.Fprintf(&bar, "[%s]▱", tagAccentEmpty)
		}
	}
	return bar.String()
}

// CreateTrackExtras renders the metadata block under the title: artist,
// album, and honest audio specs (format, sample rate, bitrate, channels, file
// size, play count). Each field on its own line as "label  value", labels
// faint and values dim, so the block reads as a definition list instead of a
// run-on string. Empty fields are omitted. No fake oscilloscope or struct dump.
func CreateTrackExtras(track domain.Song) string {
	var lines []string
	add := func(label, value string) {
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("  ["+tagFgFaint+"]%-8s ["+tagFgDim+"]%s", label, value))
	}
	add("Artist", track.Artist)
	add("Album", track.Album)
	formatStr := strings.ToUpper(track.Suffix)
	if formatStr == "" {
		formatStr = "?"
	}
	add("Format", formatStr)
	if track.SampleRate > 0 {
		add("Rate", fmt.Sprintf("%.1fkHz", float64(track.SampleRate)/1000))
	}
	if track.BitRate > 0 {
		add("Bitrate", fmt.Sprintf("%dkbps", track.BitRate))
	}
	chStr := "Mono"
	if track.ChannelCount == 2 {
		chStr = "Stereo"
	} else if track.ChannelCount > 2 {
		chStr = fmt.Sprintf("%dch", track.ChannelCount)
	}
	add("Channels", chStr)
	if track.Size > 0 {
		add("Size", fmt.Sprintf("%.1f MB", float64(track.Size)/1024/1024))
	}
	if track.PlayCount > 0 {
		add("Plays", fmt.Sprintf("%d", track.PlayCount))
	}
	return strings.Join(lines, "\n")
}

// FormatSongInfo renders the left "Now Playing" panel: an album cover
// thumbnail (when loaded), a bold title, then the per-field metadata block
// (extra). Live state lives in the bottom strip and connection in the title
// bar, so nothing is shown twice.
func FormatSongInfo(track domain.Song, panelWidth int, cover, extra string) string {
	_ = panelWidth
	coverSection := ""
	if cover != "" {
		coverSection = cover + "\n"
	}
	extraSection := ""
	if extra != "" {
		extraSection = extra + "\n"
	}
	return fmt.Sprintf(
		"\n"+
			"%s"+
			"  ["+tagFg+"::b]%s[-:-:-]\n"+
			"%s",
		coverSection,
		track.Title,
		extraSection,
	)
}

func CreateVolumeBar(volume float64, width int) string {
	if width < 5 {
		width = 5
	}
	return fmt.Sprintf("[%s]vol %s [%s]%.0f%%", tagFgFaint, blockBar(volume/100.0, width), tagFgDim, volume)
}

func CreateProgressBar(progress float64, width int) string {
	if width < 10 {
		width = 10
	}
	return blockBar(progress, width)
}

// CreateBottomBar renders the bottom transport strip. Two lines: current
// time, progress bar, total time on the first; spinner, status, and volume
// on the second. The only home for live state, so each value appears once.
// statusLabel arrives pre-colored (teal for playing, amber for paused).
func CreateBottomBar(progress float64, width int, currentTime, totalTime string, volume float64, statusLabel, spinner string) string {
	bar := CreateProgressBar(progress, width)
	volBar := CreateVolumeBar(volume, 5)
	return fmt.Sprintf(
		"\n  [%s]%s %s [%s]%s\n  [%s]%s %s                              %s\n",
		tagAccent, currentTime,
		bar,
		tagFgDim, totalTime,
		tagFgDim, spinner,
		statusLabel,
		volBar,
	)
}

// CreateIdleBottomBar renders the bottom strip when nothing is playing.
// Minimal: a Ready label and the play hint. No rule lines; whitespace
// carries the structure.
func CreateIdleBottomBar() string {
	return "\n  [" + tagFg + "]Ready\n\n  [" + tagFgFaint + "]Select a song and press [" + tagFg + "]ENTER[" + tagFgFaint + "] to play\n"
}

// keyHint renders one keybinding as a faint key aligned to 8 cells, followed by
// its action in primary text. Mirrors the label:value rhythm of the now-playing
// metadata so the welcome screen reads like the playing view, one item per line.
func keyHint(key, action string) string {
	return fmt.Sprintf("  ["+tagFgFaint+"]%-8s ["+tagFg+"]%s\n", key, action)
}

// CreateWelcomeMessage renders the idle welcome screen: app name, song count,
// then one keybinding per line. Connection state is shown in the title bar,
// not here, so it is never duplicated.
func CreateWelcomeMessage(totalSongs int) string {
	return fmt.Sprintf(
		"\n\n"+
			"  ["+tagFg+"::b]NaviCLI[-:-:-]\n"+
			"  ["+tagFgDim+"]Terminal music player\n"+
			"\n"+
			"  ["+tagFgDim+"]%d songs loaded\n"+
			"\n"+
			keyHint("/", "search")+
			keyHint("?", "help")+
			keyHint("j/k", "navigate")+
			keyHint("SPACE", "play")+
			"\n",
		totalSongs,
	)
}
