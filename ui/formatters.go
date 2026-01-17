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
	// 格式化比特率
	bitRateStr := ""
	if track.BitRate > 0 {
		bitRateStr = fmt.Sprintf("%d kbps", track.BitRate)
	}

	// 格式化文件格式
	formatStr := track.Suffix
	if formatStr == "" {
		formatStr = "unknown"
	}

	// 格式化曲目编号
	trackNumStr := ""
	if track.Track > 0 {
		trackNumStr = fmt.Sprintf("#%d", track.Track)
	}

	// 格式化采样率
	sampleRateStr := ""
	if track.SampleRate > 0 {
		sampleRateStr = fmt.Sprintf("%.1f kHz", float64(track.SampleRate)/1000)
	}

	// 组合技术信息
	techInfo := ""
	if bitRateStr != "" {
		techInfo = bitRateStr
	}
	if sampleRateStr != "" {
		if techInfo != "" {
			techInfo += " | " + sampleRateStr
		} else {
			techInfo = sampleRateStr
		}
	}
	if techInfo == "" {
		techInfo = "N/A"
	}

	return fmt.Sprintf(`
[white]Current %d:
%s

[darkgray][duration] %s [darkgray][format] %s [darkgray][size] %.1f MB
[darkgray][quality] %s

[gray]Artist: [white]%s
[gray]Album:  [white]%s %s
%s

[darkgray] SPACE (pause)
[darkgray] L/H (next/prev)
[darkgray] j/k (row)
[darkgray] J/K (page)
[darkgray] gg/G (nav)
[darkgray] +/- (volume)
[darkgray] ? (help)`,
		index+1, status, FormatDuration(track.Duration), formatStr,
		float64(track.Size)/1024/1024, techInfo,
		track.Artist, track.Album, trackNumStr, progressBar)
}

// FormatSongInfoWithCover creates a formatted info display with cover art
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
[lightgreen] Welcome to NaviCLI
[darkgray][play] Ready to Play Music!
[darkgray][source] Source: Navidrome

[darkgray]task := "[yellow]programming[darkgray]"
[darkgray][red]func[darkgray] [green]navicli[darkgray]([yellow]task[darkgray] [lightblue]string[darkgray]) [lightblue]string[darkgray] {
[darkgray]    [red]return[darkgray] "A series of mixes for listening while" [red]+[darkgray] task [red]+[darkgray] \
[darkgray]         "to focus the brain and i nspire the mind.[darkgray]"
[darkgray]}
[darkgray]
[gray]  SPACE (play/pause) |
[gray]  N/P or L/H (next/prev)
[gray]  J/K (page) | j/k (row)
[gray]  gg (start) | G (end)
[gray]  / (search) | ? (help) | Q (queue)
[gray]  ESC to exit

[darkgray]// %d songs loaded
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
