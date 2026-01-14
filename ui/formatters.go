package ui

import (
	"fmt"

	"github.com/yhkl-dev/NaviCLI/domain"
)

// FormatDuration converts seconds to MM:SS format
func FormatDuration(seconds int) string {
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// FormatSongInfo creates a formatted info display for a song
func FormatSongInfo(track domain.Song, index int, status, progressBar string) string {
	return fmt.Sprintf(`
[white]Current %d:
%s

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
		index+1, status, FormatDuration(track.Duration),
		float64(track.Size)/1024/1024, track.Artist, track.Album, track.Album, progressBar)
}

// CreateProgressBar creates a visual progress bar
func CreateProgressBar(progress float64, width int) string {
	filledWidth := int(progress * float64(width))
	var bar string

	for i := 0; i < width; i++ {
		if i < filledWidth {
			bar += "[lightgreen]▓"
		} else {
			bar += "[darkgray]░"
		}
	}
	return bar + fmt.Sprintf("[white] %.1f%%", progress*100)
}

// CreateWelcomeMessage creates the welcome screen message
func CreateWelcomeMessage(totalSongs int) string {
	return fmt.Sprintf(`
[white]Current:
[lightgreen]Welcome to NaviCLI

[darkgray][play] Ready
[darkgray][source] Navidrome
[darkgray][favourite]

[gray]Playback:
[gray]  SPACE (play/pause)
[gray]  N/P or ←/→ (prev/next)
[gray]Features:
[gray]  / (search) | ? (help) | Q (queue)
[gray]Select a track to start
[gray]ESC to exit

[darkgray][red]func[darkgray] [green]navicli[darkgray]([yellow]task[darkgray] [lightblue]string[darkgray]) [lightblue]string[darkgray] {
[darkgray]    [red]return[darkgray] "^A series of mixes for listening while" [red]+[darkgray] task [red]+[darkgray] \
[darkgray]         "to focus the brain and i nspire the mind.[darkgray]"
[darkgray]}
[darkgray]
[darkgray]task := "[yellow]programming[darkgray]"

[darkgray]// %d songs loaded
[darkgray]// Search, Queue, Help - All ready
[darkgray]// Written by github.com/yhkl-dev
[darkgray]// Auto-play next enabled`, totalSongs)
}

// CreateIdleDisplay creates the idle state display
func CreateIdleDisplay() string {
	return `
[darkgray][invert] [darkgray][fullscreen]`
}

// CreateProgressText creates the progress time display
func CreateProgressText(currentTime, totalTime, volumeText string) string {
	return fmt.Sprintf(`
[darkgray]%s/%s [darkgray][v-] [white]%s [darkgray][v+] [darkgray][random]`, currentTime, totalTime, volumeText)
}
