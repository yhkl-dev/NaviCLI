package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/viper"
	"github.com/wildeyedskies/go-mpv/mpv"
	"github.com/yhkl-dev/NaviCLI/mpvplayer"
	"github.com/yhkl-dev/NaviCLI/subsonic"
)

func formatDuration(seconds int) string {
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

type Application struct {
	application    *tview.Application
	subsonicClient *subsonic.Client
	mpvInstance    *mpvplayer.Mpvplayer
	totalSongs     []subsonic.Song
	currentPage    int
	pageSize       int
	totalPages     int

	rootFlex    *tview.Flex
	songTable   *tview.Table
	statusBar   *tview.TextView
	progressBar *tview.TextView
	statsBar    *tview.TextView
	currentSong *subsonic.Song
	isPlaying   bool
}

func (a *Application) setupPagination() {
	a.pageSize = 20
	a.currentPage = 1
}

func (a *Application) getCurrentPageData() []subsonic.Song {
	start := (a.currentPage - 1) * a.pageSize
	end := min(start+a.pageSize, len(a.totalSongs))
	return a.totalSongs[start:end]
}

func (a *Application) updateProgressBar() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if a.mpvInstance == nil || !a.isPlaying {
			continue
		}

		pos, err := a.mpvInstance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
		if err != nil {
			continue
		}
		duration, err := a.mpvInstance.GetProperty("duration", mpv.FORMAT_DOUBLE)
		if err != nil {
			continue
		}

		currentPos := pos.(float64)
		totalDuration := duration.(float64)

		var progress float64
		if totalDuration > 0 {
			progress = currentPos / totalDuration
		}

		const barWidth = 50
		filled := int(progress * barWidth)
		empty := barWidth - filled

		progressBar := "[green]"
		for range filled {
			progressBar += "━"
		}
		progressBar += "[white]"
		for range empty {
			progressBar += "━"
		}

		currentTime := formatDuration(int(currentPos))
		totalTime := formatDuration(int(totalDuration))

		a.application.QueueUpdateDraw(func() {
			a.progressBar.SetText(fmt.Sprintf("%s %s / %s", progressBar, currentTime, totalTime))
		})
	}
}

func (a *Application) updateSongInfo() {
	if a.currentSong == nil {
		return
	}

	song := a.currentSong
	info := fmt.Sprintf("[yellow]Artist: [white]%s\n"+
		"[yellow]Album: [white]%s\n"+
		"[yellow]Title: [white]%s\n"+
		"[yellow]Bitrate: [white]%d kbps\n"+
		"[yellow]Size: [white]%.2f MB",
		song.Artist,
		song.Album,
		song.Title,
		song.BitRate,
		float64(song.Size)/1024/1024)

	a.application.QueueUpdateDraw(func() {
		a.statsBar.SetText(info)
	})
}

func (a *Application) prevPage() {
	if a.currentPage <= 1 {
		return
	}
	a.currentPage--
	a.application.QueueUpdateDraw(func() {
		a.renderSongTable()
		a.application.SetFocus(a.songTable)

	})
}

func (a *Application) nextPage() {
	fmt.Println("nextPage")
	if a.currentPage >= a.totalPages {
		return
	}
	a.currentPage++
	a.application.QueueUpdateDraw(func() {
		a.renderSongTable()
		a.application.SetFocus(a.songTable)
	})
}

func (a *Application) createHomepage() {
	a.progressBar = tview.NewTextView().
		SetDynamicColors(true)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true)

	a.statsBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)

	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	a.songTable.SetCell(0, 0, tview.NewTableCell("#").SetTextColor(tcell.ColorYellow))
	a.songTable.SetCell(0, 1, tview.NewTableCell("Song").SetTextColor(tcell.ColorYellow))
	a.songTable.SetCell(0, 2, tview.NewTableCell("Artist").SetTextColor(tcell.ColorYellow))
	a.songTable.SetCell(0, 3, tview.NewTableCell("Album").SetTextColor(tcell.ColorYellow))
	a.songTable.SetCell(0, 4, tview.NewTableCell("Duration").SetTextColor(tcell.ColorYellow))

	a.songTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(a.totalSongs) {
			currentTrack := a.totalSongs[row-1]
			playURL := a.subsonicClient.GetPlayURL(currentTrack.ID)
			a.mpvInstance.Queue = append(a.mpvInstance.Queue, mpvplayer.QueueItem{
				Id:       currentTrack.ID,
				Uri:      playURL,
				Title:    currentTrack.Title,
				Artist:   currentTrack.Artist,
				Duration: currentTrack.Duration,
			})

			a.mpvInstance.Play(playURL)
			a.isPlaying = true
			a.statusBar.SetText(fmt.Sprintf("[green]Playing: %s - %s",
				currentTrack.Title,
				currentTrack.Artist,
			))
			a.currentSong = &currentTrack
			go a.updateSongInfo()
		}
	})

	bottomFlex := tview.NewFlex().
		AddItem(a.statusBar, 0, 1, false).
		AddItem(a.statsBar, 50, 0, false)

	a.rootFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.songTable, 0, 1, true).
		AddItem(a.progressBar, 1, 0, false).
		AddItem(bottomFlex, 5, 0, false)

	a.application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			if a.isPlaying {
				a.mpvInstance.Pause()
				a.isPlaying = false
				a.statusBar.SetText(fmt.Sprintf("[yellow]Paused: %s - %s",
					a.currentSong.Title,
					a.currentSong.Artist,
				))
			} else {
				a.mpvInstance.Pause()
				a.isPlaying = true
				a.statusBar.SetText(fmt.Sprintf("[green]Playing: %s - %s",
					a.currentSong.Title,
					a.currentSong.Artist,
				))
			}
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyENQ:
			a.application.Stop()
		}
		return event
	})
	a.application.SetRoot(a.rootFlex, true)

}

func (a *Application) renderSongTable() {
	for i := a.songTable.GetRowCount() - 1; i > 0; i-- {
		a.songTable.RemoveRow(i)
	}
	pageData := a.getCurrentPageData()
	cells := make([]*tview.TableCell, 0, len(pageData)*5)
	for i, song := range pageData {
		cells = append(cells,
			tview.NewTableCell(fmt.Sprintf("%d", i+1)).SetAlign(tview.AlignRight),
			tview.NewTableCell(song.Title).SetExpansion(1).SetMaxWidth(40),
			tview.NewTableCell(song.Artist).SetMaxWidth(40),
			tview.NewTableCell(song.Album).SetMaxWidth(40),
			tview.NewTableCell(formatDuration(song.Duration)).SetAlign(tview.AlignRight).SetMaxWidth(40),
		)
	}

	for i := range pageData {
		for col := range 5 {
			a.songTable.SetCell(i+1, col, cells[i*5+col])
		}
	}
}

func (a *Application) loadMusic() error {
	songs, err := a.subsonicClient.GetPlaylists()
	if err != nil {
		return fmt.Errorf("获取歌曲列表失败: %v", err)
	}

	if !reflect.DeepEqual(a.totalSongs, songs) {
		a.totalSongs = songs
		a.totalPages = (len(a.totalSongs) + a.pageSize - 1) / a.pageSize
		a.application.QueueUpdateDraw(func() {
			a.renderSongTable()
		})
	}
	return nil
}

func eventListener(ctx context.Context, m *mpv.Mpv) chan *mpv.Event {
	c := make(chan *mpv.Event)
	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				e := m.WaitEvent(1)
				c <- e
			}
		}
	}()
	return c
}

func ViperInit() {
	required := []string{
		"server.url",
		"server.username",
		"server.password",
	}
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("fatal error config file: %w \n", err)
	}

	for _, key := range required {
		if !viper.IsSet(key) {
			log.Fatalf("missing required config: %s", key)
		}
	}
}

func main() {
	ViperInit()

	subsonicClient := subsonic.Init(
		viper.GetString("server.url"),
		viper.GetString("server.username"),
		viper.GetString("server.password"),
		"goplayer",
		"1.16.1",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mpvInstance, err := mpvplayer.CreateMPVInstance()
	if err != nil {
		log.Fatal(err)
	}

	app := &Application{
		application:    tview.NewApplication(),
		subsonicClient: subsonicClient,
		mpvInstance:    &mpvplayer.Mpvplayer{mpvInstance, eventListener(ctx, mpvInstance), make([]mpvplayer.QueueItem, 0), false},
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
		app.application.Stop()
	}()
	app.setupPagination()
	app.createHomepage()
	go app.updateProgressBar()
	go func() {
		if err := app.loadMusic(); err != nil {
			app.application.QueueUpdateDraw(func() {
				app.statusBar.SetText("[red]加载失败: " + err.Error())
			})
		}
	}()

	if err := app.application.Run(); err != nil {
		log.Fatal(err)
	}
}
