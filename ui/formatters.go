package ui

import (
	"fmt"
	"strings"

	"github.com/yhkl-dev/NaviCLI/domain"
)

var brailleBits = [8]uint{0x01, 0x02, 0x04, 0x40, 0x08, 0x10, 0x20, 0x80}

var spinnerFrames = []string{"◴", "◷", "◶", "◵"}

func brailleForFill(n int) rune {
	if n <= 0 {
		return 0x2800
	}
	if n > 8 {
		n = 8
	}
	var code uint
	for i := 0; i < n; i++ {
		code |= brailleBits[i]
	}
	return rune(0x2800 + code)
}

func SpinnerChar(tick int) string {
	return spinnerFrames[tick%len(spinnerFrames)]
}

func FormatDuration(seconds int) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ---- Geek paused extras ----

var specBars = []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

func CreateFrozenSpectrum(width int) string {
	bars := width - 4
	if bars > 40 {
		bars = 40
	}
	if bars < 10 {
		bars = 10
	}

	var row1, row2, row3 string
	for i := 0; i < bars; i++ {
		phase1 := (i * 7) % 32
		phase2 := (i*11 + 8) % 32
		phase3 := (i*5 + 16) % 32

		v1 := 8 - abs(phase1-16)/2
		v2 := 8 - abs(phase2-16)/2
		v3 := 8 - abs(phase3-16)/2
		if v1 < 0 { v1 = 0 }
		if v2 < 0 { v2 = 0 }
		if v3 < 0 { v3 = 0 }

		row1 += specBars[v1]
		row2 += specBars[v2]
		row3 += specBars[v3]
	}

	return fmt.Sprintf("  [#ffb300]%s\n  [#e65100]%s\n  [#5d4037]%s", row1, row2, row3)
}

func CreateOscilloscope(width int) string {
	bars := width - 4
	if bars > 40 { bars = 40 }
	if bars < 12 { bars = 12 }

	var row1, row2, row3 strings.Builder
	for i := 0; i < bars; i++ {
		phase := float64(i) * 0.5
		// Composite sine: fundamental + 3rd harmonic
		v := 3.0*sinApprox(phase) + 1.5*sinApprox(phase*3.0+1.0)
		level := int((v+4.5)/9.0*7.0 + 0.5)
		if level < 0 { level = 0 }
		if level > 7 { level = 7 }

		// Build 3-row column based on level
		switch {
		case level <= 1:
			row1.WriteString(" ")
			row2.WriteString(" ")
			row3.WriteRune('─')
		case level <= 2:
			row1.WriteString(" ")
			row2.WriteRune('╰')
			row3.WriteRune('╭')
		case level <= 3:
			row1.WriteRune('╭')
			row2.WriteRune('│')
			row3.WriteRune('╰')
		case level <= 4:
			row1.WriteRune('╭')
			row2.WriteRune('│')
			row3.WriteRune('│')
		case level <= 5:
			row1.WriteRune('╭')
			row2.WriteRune('│')
			row3.WriteRune('╯')
		case level <= 6:
			row1.WriteRune('╭')
			row2.WriteRune('╰')
			row3.WriteString(" ")
		default:
			row1.WriteRune('╮')
			row2.WriteString(" ")
			row3.WriteString(" ")
		}
	}

	return fmt.Sprintf("  [#ffb300]%s\n  [#e65100]%s\n  [#5d4037]%s",
		row1.String(), row2.String(), row3.String())
}

func sinApprox(x float64) float64 {
	// Bhaskara I sine approximation: accurate enough for visuals
	x = x - float64(int(x/6.283185307)*6) // mod 2π
	if x < 0 { x += 6.283185307 }
	if x > 3.141592653 {
		return -sinApprox(x - 3.141592653)
	}
	return 16.0*x*(3.141592653-x)/(5.0*3.141592653*3.141592653 - 4.0*x*(3.141592653-x))
}

func CreateAudioSpecs(track domain.Song, width int) string {
	formatStr := strings.ToUpper(track.Suffix)
	if formatStr == "" {
		formatStr = "?"
	}

	sampleStr := ""
	if track.SampleRate > 0 {
		sampleStr = fmt.Sprintf(" · %.1fkHz", float64(track.SampleRate)/1000)
	}

	bitrateStr := ""
	if track.BitRate > 0 {
		bitrateStr = fmt.Sprintf(" · %dkbps", track.BitRate)
	}

	chStr := "Mono"
	if track.ChannelCount == 2 {
		chStr = "Stereo"
	} else if track.ChannelCount > 2 {
		chStr = fmt.Sprintf("%dch", track.ChannelCount)
	}

	sizeStr := ""
	if track.Size > 0 {
		sizeStr = fmt.Sprintf(" · %.1f MB", float64(track.Size)/1024/1024)
	}

	playStr := ""
	if track.PlayCount > 0 {
		playStr = fmt.Sprintf(" · %d plays", track.PlayCount)
	}

	line1 := fmt.Sprintf("%s%s%s", formatStr, sampleStr, bitrateStr)
	line2 := fmt.Sprintf("%s%s%s", chStr, sizeStr, playStr)

	return fmt.Sprintf("  [#ffb300]%s\n  [gray]%s", line1, line2)
}

func CreateHexDump(track domain.Song, width int) string {
	if width < 36 {
		return CreateGoDebug(track)
	}

	title := track.Title
	artist := track.Artist
	formatStr := strings.ToUpper(track.Suffix)

	lines := []string{
		hexLine(0x0000, []byte(title)),
		hexLine(0x0010, []byte(artist)),
	}

	specData := fmt.Sprintf("%s %dHz %dkbps ch=%d", formatStr, track.SampleRate, track.BitRate, track.ChannelCount)
	lines = append(lines, hexLine(0x0020, []byte(specData)))

	var result string
	for _, l := range lines {
		result += "  [gray]" + l + "\n"
	}
	return strings.TrimRight(result, "\n")
}

func hexLine(addr int, data []byte) string {
	hexPart := make([]byte, 0, 48)
	for i, b := range data {
		if i > 0 && i%8 == 0 {
			hexPart = append(hexPart, ' ')
		}
		hexPart = append(hexPart, fmt.Sprintf("%02X ", b)...)
	}

	asciiPart := make([]byte, 0, 16)
	for _, b := range data {
		if b >= 32 && b < 127 {
			asciiPart = append(asciiPart, b)
		} else {
			asciiPart = append(asciiPart, '.')
		}
	}

	return fmt.Sprintf("%04X  %-24s %s", addr, string(hexPart), string(asciiPart))
}

func CreateGoDebug(track domain.Song) string {
	formatStr := strings.ToUpper(track.Suffix)
	if formatStr == "" {
		formatStr = "?"
	}

	return fmt.Sprintf(
		"  [darkgray]type Song struct {\n"+
			"  [gray]    Title:      [white]%q\n"+
			"  [gray]    Artist:     [white]%q\n"+
			"  [gray]    Format:     [white]%q\n"+
			"  [gray]    BitRate:    [white]%d\n"+
			"  [gray]    SampleRate: [white]%d\n"+
			"  [gray]    Channels:   [white]%d\n"+
			"  [gray]    Size:       [white]%.1f MB\n"+
			"  [darkgray]}",
		track.Title,
		track.Artist,
		formatStr,
		track.BitRate,
		track.SampleRate,
		track.ChannelCount,
		float64(track.Size)/1024/1024,
	)
}

func CreatePlayingExtras(track domain.Song, width int) string {
	parts := []string{
		CreateOscilloscope(width),
		"",
		CreateAudioSpecs(track, width),
	}
	return strings.Join(parts, "\n")
}

func CreatePausedExtras(track domain.Song, width int) string {
	parts := []string{
		CreateOscilloscope(width),
		"",
		CreateAudioSpecs(track, width),
		"",
		CreateGoDebug(track),
	}
	return strings.Join(parts, "\n")
}

// ---- Song info / progress bar / welcome ----

func FormatSongInfo(track domain.Song, status string, spinner string, volumeBar string, panelWidth int, connected bool, extra string) string {
	duration := FormatDuration(track.Duration)

	trackInfo := ""
	if track.Track > 0 {
		trackInfo = fmt.Sprintf("Track #%d  ", track.Track)
	}

	sepWidth := panelWidth - 2
	if sepWidth < 10 {
		sepWidth = 10
	}
	sep := strings.Repeat("─", sepWidth)

	connDot := "[darkgray]● [gray]Disconnected"
	if connected {
		connDot = "[green]● [gray]Navidrome connected"
	}

	extraSection := ""
	if extra != "" {
		extraSection = extra + "\n"
	}

	return fmt.Sprintf(
		"\n"+
			"  %s [white]%s\n"+
			"  [gray]%s\n"+
			"  [gray]%s  %s· %s\n"+
			"  [darkgray]%s\n"+
			"%s"+
			"  %s\n"+
			"  %s\n"+
			"\n"+
			"  %s\n",
		spinner,
		track.Title,
		track.Artist,
		track.Album,
		trackInfo,
		duration,
		sep,
		extraSection,
		volumeBar,
		connDot,
		status,
	)
}

func CreateVolumeBar(volume float64, width int) string {
	if width < 6 {
		width = 6
	}
	filled := int(volume / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var bar string
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "[#ffb300]█"
		} else {
			bar += "[#424242]░"
		}
	}
	return fmt.Sprintf("[darkgray]Vol: %s [white]%.0f%%", bar, volume)
}

func CreateProgressBar(progress float64, width int) string {
	if width < 10 {
		width = 10
	}

	totalDots := width * 8
	filledDots := int(progress * float64(totalDots))
	if filledDots > totalDots {
		filledDots = totalDots
	}
	if filledDots < 0 {
		filledDots = 0
	}

	var bar string
	for i := 0; i < width; i++ {
		dotsBefore := i * 8
		charFilled := 0
		if filledDots > dotsBefore {
			charFilled = filledDots - dotsBefore
			if charFilled > 8 {
				charFilled = 8
			}
		}

		charRatio := float64(dotsBefore) / float64(totalDots)
		switch {
		case charFilled == 8:
			bar += fmt.Sprintf("[#ffb300]%c", brailleForFill(8))
		case charFilled > 0:
			bar += fmt.Sprintf("[#e65100]%c", brailleForFill(charFilled))
		case charRatio < progress+0.05:
			bar += fmt.Sprintf("[#5d4037]%c", brailleForFill(0))
		default:
			bar += fmt.Sprintf("[#424242]%c", brailleForFill(0))
		}
	}
	return bar
}

func CreateBottomBar(progress float64, width int, currentTime, totalTime, volumeText, statusLabel string, spinner string) string {
	bar := CreateProgressBar(progress, width)
	pct := fmt.Sprintf("[white]%.1f%%", progress*100)

	return fmt.Sprintf(
		"\n  %s %s\n  [gray]%s  [darkgray]── %s ──  [gray]%s    [darkgray]Vol: [white]%s\n  %s %s\n",
		bar, pct,
		currentTime, spinner, totalTime, volumeText,
		spinner, statusLabel,
	)
}

func CreateIdleBottomBar() string {
	return fmt.Sprintf(
		"\n  [darkgray]── [gray]Ready [darkgray]───────────────────────────────────────────────\n" +
			"  [gray]Select a song and press [white]ENTER[gray] to play\n" +
			"  [darkgray]──────────────────────────────────────────────────────\n",
	)
}

func CreateWelcomeMessage(totalSongs int) string {
	return fmt.Sprintf(
		"\n\n"+
			"  [white]NaviCLI\n"+
			"  [gray]Terminal Music Player\n"+
			"\n"+
			"  [gray]%d songs loaded\n"+
			"  [gray]Navidrome connected\n"+
			"\n"+
			"  [darkgray]/ [white]search  [darkgray]? [white]help  [darkgray]q [white]queue\n"+
			"  [darkgray]j/k [white]navigate  [darkgray]SPACE [white]play\n"+
			"\n",
		totalSongs,
	)
}
