package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
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
	isLoading   bool // 添加加载状态锁
	loadingMux  sync.Mutex // 互斥锁保护加载状态
	
	// 当前播放索引
	currentSongIndex int
}

func (a *Application) setupPagination() {
	a.pageSize = 500
	a.currentPage = 1
	a.currentSongIndex = -1 // 初始化为-1表示没有歌曲正在播放
	a.isLoading = false     // 初始化加载状态
}

// 播放指定索引的歌曲
func (a *Application) playSongAtIndex(index int) {
	if index < 0 || index >= len(a.totalSongs) {
		return
	}
	
	// 使用互斥锁确保线程安全
	a.loadingMux.Lock()
	if a.isLoading {
		log.Printf("正在加载中，跳过播放请求: %d", index)
		a.loadingMux.Unlock()
		return
	}
	// 设置加载状态
	a.isLoading = true
	// 立即更新当前歌曲状态，避免UI显示不一致
	a.currentSongIndex = index
	currentTrack := a.totalSongs[index]
	a.currentSong = &currentTrack
	// 重置播放状态
	a.isPlaying = false
	a.loadingMux.Unlock()
	
	log.Printf("开始加载歌曲: %d", index)
	
	// 显示加载中状态 - 使用最新的歌曲信息
	loadingBar := "[yellow]▓▓▓[darkgray]░░░░░░░░░░░░░░░░░░░ Loading..."
	info := fmt.Sprintf(`
[white]Episode %d:
[yellow]%s [darkgray](Loading...)

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
		index+1,
		currentTrack.Title,
		formatDuration(currentTrack.Duration),
		float64(currentTrack.Size)/1024/1024,
		currentTrack.Artist,
		currentTrack.Album,
		currentTrack.Album,
		loadingBar)
	
	// 立即更新UI显示加载状态
	a.application.QueueUpdateDraw(func() {
		if a.statusBar != nil {
			a.statusBar.SetText(info)
		}
	})
	
	// 异步处理播放，但要同步状态更新
	go func() {
		defer func() {
			// 无论成功还是失败，都要释放加载锁
			a.loadingMux.Lock()
			a.isLoading = false
			a.loadingMux.Unlock()
			log.Printf("加载完成，释放锁: %d", index)
			
			if r := recover(); r != nil {
				log.Printf("播放歌曲时出错: %v", r)
				// 播放失败，更新状态为失败
				a.isPlaying = false
				a.application.QueueUpdateDraw(func() {
					if a.statusBar != nil {
						failedInfo := fmt.Sprintf(`
[white]Episode %d:
[red]%s [darkgray](Failed)

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
[red]播放失败`,
							index+1,
							currentTrack.Title,
							formatDuration(currentTrack.Duration),
							float64(currentTrack.Size)/1024/1024,
							currentTrack.Artist,
							currentTrack.Album,
							currentTrack.Album)
						a.statusBar.SetText(failedInfo)
					}
				})
			}
		}()
		
		// 获取播放URL
		log.Printf("开始获取播放URL: %s", currentTrack.Title)
		
		// 设置超时的网络请求
		done := make(chan string, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("获取URL时出错: %v", r)
					done <- ""
				}
			}()
			url := a.subsonicClient.GetPlayURL(currentTrack.ID)
			done <- url
		}()
		
		var playURL string
		select {
		case playURL = <-done:
			if playURL == "" {
				log.Printf("获取播放URL失败: %s", currentTrack.Title)
				return
			}
		case <-time.After(10 * time.Second):
			log.Printf("获取播放URL超时: %s", currentTrack.Title)
			return
		}
		
		log.Printf("获取到播放URL: %s - %s", currentTrack.Artist, currentTrack.Title)
		
		// 更新队列
		if a.mpvInstance != nil {
			a.mpvInstance.Queue = []mpvplayer.QueueItem{{
				Id:       currentTrack.ID,
				Uri:      playURL,
				Title:    currentTrack.Title,
				Artist:   currentTrack.Artist,
				Duration: currentTrack.Duration,
			}}

			// 停止当前播放
			if a.mpvInstance.Mpv != nil {
				a.mpvInstance.Stop()
				time.Sleep(50 * time.Millisecond)
			}
			
			// 开始播放
			if a.mpvInstance.Mpv != nil {
				a.mpvInstance.Play(playURL)
				
				// 播放命令发送成功后，更新为播放状态
				a.isPlaying = true
				
				// 更新UI为播放状态
				playingBar := "[lightgreen]▓[darkgray]░░░░░░░░░░░░░░░░░░░ 0.0%"
				playingInfo := fmt.Sprintf(`
[white]Episode %d:
[lightgreen]%s

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
					index+1,
					currentTrack.Title,
					formatDuration(currentTrack.Duration),
					float64(currentTrack.Size)/1024/1024,
					currentTrack.Artist,
					currentTrack.Album,
					currentTrack.Album,
					playingBar)
				
				a.application.QueueUpdateDraw(func() {
					if a.statusBar != nil {
						a.statusBar.SetText(playingInfo)
					}
				})
				
				log.Printf("播放开始: %s", currentTrack.Title)
				
				// 稍作等待，让播放状态稳定，避免被进度更新立即覆盖
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
}

// 播放下一首歌曲
func (a *Application) playNextSong() {
	if len(a.totalSongs) == 0 {
		return
	}
	
	// 使用互斥锁检查是否正在加载
	a.loadingMux.Lock()
	isCurrentlyLoading := a.isLoading
	a.loadingMux.Unlock()
	
	if isCurrentlyLoading {
		log.Printf("正在加载中，跳过下一首请求")
		return
	}
	
	nextIndex := a.currentSongIndex + 1
	if nextIndex >= len(a.totalSongs) {
		nextIndex = 0 // 循环到第一首
	}
	
	log.Printf("切换到下一首: %d -> %d", a.currentSongIndex, nextIndex)
	// 异步执行播放，但在主线程已经检查了锁
	go a.playSongAtIndex(nextIndex)
}

// 播放上一首歌曲
func (a *Application) playPreviousSong() {
	if len(a.totalSongs) == 0 {
		return
	}
	
	// 使用互斥锁检查是否正在加载
	a.loadingMux.Lock()
	isCurrentlyLoading := a.isLoading
	a.loadingMux.Unlock()
	
	if isCurrentlyLoading {
		log.Printf("正在加载中，跳过上一首请求")
		return
	}
	
	prevIndex := a.currentSongIndex - 1
	if prevIndex < 0 {
		prevIndex = len(a.totalSongs) - 1 // 循环到最后一首
	}
	
	log.Printf("切换到上一首: %d -> %d", a.currentSongIndex, prevIndex)
	// 异步执行播放，但在主线程已经检查了锁
	go a.playSongAtIndex(prevIndex)
}

func (a *Application) getCurrentPageData() []subsonic.Song {
	start := (a.currentPage - 1) * a.pageSize
	end := min(start+a.pageSize, len(a.totalSongs))
	return a.totalSongs[start:end]
}

func (a *Application) updateProgressBar() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 添加安全检查
			if a.application == nil || a.mpvInstance == nil {
				return
			}

			// 检查加载状态，如果正在加载则跳过更新
			a.loadingMux.Lock()
			isCurrentlyLoading := a.isLoading
			currentSongPtr := a.currentSong
			currentIndex := a.currentSongIndex
			isCurrentlyPlaying := a.isPlaying
			a.loadingMux.Unlock()
			
			if isCurrentlyLoading {
				// 正在加载中，不更新进度条，避免覆盖加载状态
				continue
			}

			if a.mpvInstance.Mpv == nil {
				// 空闲状态显示 - NCW风格
				a.application.QueueUpdateDraw(func() {
					if a.progressBar != nil {
						idleDisplay := `
[darkgray][about] [darkgray][credits] [darkgray][rss.xml]
[darkgray][patreon] [darkgray][podcasts.apple]
[darkgray][folder.jpg] [darkgray][enterprise mode]
[darkgray][invert] [darkgray][fullscreen]`
						a.progressBar.SetText(idleDisplay)
					}
				})
				continue
			}

			if !isCurrentlyPlaying {
				// 暂停状态显示 - 显示进度条
				if currentSongPtr != nil {
					a.application.QueueUpdateDraw(func() {
						if a.progressBar != nil && a.statusBar != nil {
							pausedDisplay := `
[darkgray][prev] [darkgray][-30] [yellow][play] [darkgray][+30] [darkgray][next]
[darkgray]00:00:00 [darkgray][v-] [darkgray]100% [darkgray][v+] [darkgray][random]`
							a.progressBar.SetText(pausedDisplay)
							
							// 在状态栏显示进度条
							progressBar := "[darkgray]▓▓▓▓▓▓▓▓░░░░░░░░░░░░ 0%"
							statusInfo := fmt.Sprintf(`
[white]Episode %d:
[yellow]%s [darkgray](PAUSED)

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
								currentIndex+1,
								currentSongPtr.Title,
								formatDuration(currentSongPtr.Duration),
								float64(currentSongPtr.Size)/1024/1024,
								currentSongPtr.Artist,
								currentSongPtr.Album,
								currentSongPtr.Album,
								progressBar)
							a.statusBar.SetText(statusInfo)
						}
					})
				}
				continue
			}

			// 异步获取播放状态，完全不阻塞主线程
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// 静默处理错误，避免日志泛滥
					}
				}()
				
				// 快速检查
				if a.mpvInstance == nil || a.mpvInstance.Mpv == nil {
					return
				}
				
				// 设置更短的超时
				done := make(chan struct{})
				var currentPos, totalDuration float64
				var hasError bool
				
				go func() {
					defer func() {
						if r := recover(); r != nil {
							hasError = true
						}
						close(done)
					}()
					
					pos, err := a.mpvInstance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
					if err != nil {
						hasError = true
						return
					}
					duration, err := a.mpvInstance.GetProperty("duration", mpv.FORMAT_DOUBLE)
					if err != nil {
						hasError = true
						return
					}
					currentPos = pos.(float64)
					totalDuration = duration.(float64)
				}()
				
				select {
				case <-done:
					if hasError {
						return
					}
				case <-time.After(200 * time.Millisecond): // 大幅缩短超时时间
					return // 快速超时，避免阻塞
				}

				// 安全检查获取到的数据
				if totalDuration <= 0 || currentPos < 0 {
					return
				}

				currentTime := formatDuration(int(currentPos))
				totalTime := formatDuration(int(totalDuration))
				
				// 计算进度百分比
				progress := currentPos / totalDuration
				if progress > 1 {
					progress = 1
				} else if progress < 0 {
					progress = 0
				}
				
				// 生成进度条
				progressBarWidth := 20
				filledWidth := int(progress * float64(progressBarWidth))
				progressBar := ""
				
				for i := 0; i < progressBarWidth; i++ {
					if i < filledWidth {
						progressBar += "[lightgreen]▓"
					} else {
						progressBar += "[darkgray]░"
					}
				}
				progressBar += fmt.Sprintf("[white] %.1f%%", progress*100)
				
				// NCW风格的控制栏 - 简洁版本
				progressText := fmt.Sprintf(`
[darkgray][prev] [darkgray][-30] [lightgreen][play] [darkgray][+30] [darkgray][next]
[darkgray]%s/%s [darkgray][v-] [darkgray]100%% [darkgray][v+] [darkgray][random]`,
					currentTime, totalTime)

				// 非阻塞UI更新
				select {
				case <-time.After(10 * time.Millisecond):
					return // 如果UI更新队列繁忙，直接跳过
				default:
					a.application.QueueUpdateDraw(func() {
						if a.progressBar != nil {
							a.progressBar.SetText(progressText)
						}
						
						// 在状态栏显示进度条 - 使用正确的Episode编号
						if currentSongPtr != nil && a.statusBar != nil {
							statusInfo := fmt.Sprintf(`
[white]Episode %d:
[lightgreen]%s

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
%s`,
								currentIndex+1,
								currentSongPtr.Title,
								formatDuration(currentSongPtr.Duration),
								float64(currentSongPtr.Size)/1024/1024,
								currentSongPtr.Artist,
								currentSongPtr.Album,
								currentSongPtr.Album,
								progressBar)
							a.statusBar.SetText(statusInfo)
						}
					})
				}
			}()
			
		case <-time.After(15 * time.Second):
			// 定期检查是否应该退出
			if a.application == nil {
				return
			}
		}
	}
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
	// 创建简洁的播放进度条
	a.progressBar = tview.NewTextView().
		SetDynamicColors(true)
	a.progressBar.SetBorder(false)

	// 创建现在播放状态显示区
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true)
	a.statusBar.SetBorder(false)

	// 统计信息面板（预留）
	a.statsBar = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	a.statsBar.SetBorder(false)

	// 创建极简的歌曲列表
	a.songTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// 无边框设计
	a.songTable.SetBorder(false)

	// 设计极简的表头
	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Attributes(tcell.AttrBold)
	
	a.songTable.SetCell(0, 0, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 1, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 2, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 3, tview.NewTableCell("").
		SetStyle(headerStyle))
	a.songTable.SetCell(0, 4, tview.NewTableCell("").
		SetStyle(headerStyle))

	a.songTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(a.totalSongs) {
			// 使用互斥锁检查是否正在加载
			a.loadingMux.Lock()
			isCurrentlyLoading := a.isLoading
			a.loadingMux.Unlock()
			
			if isCurrentlyLoading {
				log.Printf("正在加载中，跳过选择播放请求")
				return
			}
			// 异步调用播放函数，但在主线程已经检查了锁
			go a.playSongAtIndex(row - 1)
		}
	})

	// 创建三栏布局 - 参考NCW风格
	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.statusBar, 0, 1, false)

	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.songTable, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftPanel, 0, 1, false).
		AddItem(rightPanel, 0, 2, true)

	a.rootFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(a.progressBar, 3, 0, false)

	a.application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case ' ':
				// 空格键：播放/暂停
				if a.mpvInstance == nil || a.mpvInstance.Mpv == nil {
					return nil
				}
				
				// 异步处理播放/暂停，避免阻塞UI
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("播放/暂停操作时出错: %v", r)
						}
					}()
					
					if a.isPlaying {
						a.mpvInstance.Pause()
						a.isPlaying = false
						// 简化暂停状态显示，不获取进度
						if a.currentSong != nil {
							info := fmt.Sprintf(`
[white]Episode %d:
[yellow]%s [darkgray](PAUSED)

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
[darkgray]▓▓▓▓▓▓▓▓░░░░░░░░░░░░ --%%`,
								a.currentSongIndex+1,
								a.currentSong.Title,
								formatDuration(a.currentSong.Duration),
								float64(a.currentSong.Size)/1024/1024,
								a.currentSong.Artist,
								a.currentSong.Album,
								a.currentSong.Album)
							
							a.application.QueueUpdateDraw(func() {
								if a.statusBar != nil {
									a.statusBar.SetText(info)
								}
							})
						}
					} else {
						a.mpvInstance.Pause()
						a.isPlaying = true
						// 简化播放状态显示，不获取进度
						if a.currentSong != nil {
							info := fmt.Sprintf(`
[white]Episode %d:
[lightgreen]%s

[darkgray][play] %s
[darkgray][source] %.1f MB
[darkgray][favourite]

[gray]%s - %s
[gray]%s
[lightgreen]▓▓▓▓▓▓▓▓░░░░░░░░░░░░ --%%`,
								a.currentSongIndex+1,
								a.currentSong.Title,
								formatDuration(a.currentSong.Duration),
								float64(a.currentSong.Size)/1024/1024,
								a.currentSong.Artist,
								a.currentSong.Album,
								a.currentSong.Album)
							
							a.application.QueueUpdateDraw(func() {
								if a.statusBar != nil {
									a.statusBar.SetText(info)
								}
							})
						}
					}
				}()
				return nil
			case 'n', 'N':
				// n键：下一首
				a.playNextSong()
				return nil
			case 'p', 'P':
				// p键：上一首
				a.playPreviousSong()
				return nil
			}
		}

		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			// ESC或Ctrl+C退出
			log.Println("用户请求退出程序")
			
			// 停止MPV播放
			if a.mpvInstance != nil && a.mpvInstance.Mpv != nil {
				a.mpvInstance.Command([]string{"quit"})
			}
			
			a.application.Stop()
			
			// 确保程序能退出
			go func() {
				time.Sleep(1 * time.Second)
				os.Exit(0)
			}()
			return nil
		case tcell.KeyRight:
			// 右箭头：下一首
			a.playNextSong()
			return nil
		case tcell.KeyLeft:
			// 左箭头：上一首
			a.playPreviousSong()
			return nil
		}
		return event
	})
	a.application.SetRoot(a.rootFlex, true)

	// 设置简洁的欢迎界面 - NCW风格
	welcomeMsg := fmt.Sprintf(`
[white]Episode 1:
[lightgreen]Welcome to NaviCLI

[darkgray][play] Ready
[darkgray][source] Music Player
[darkgray][favourite]

[gray]Press SPACE to play/pause
[gray]Press N/P or ←/→ for prev/next
[gray]Press ESC to exit
[gray]Select a track to start

[darkgray]W • Function musicFor([yellow]task[darkgray] = '[yellow]programming[darkgray]') { [yellow]return[darkgray] '^A series of mi
[darkgray]xes intended for listening while
[darkgray]$[yellow]{task}[darkgray] to focus the brain and i
[darkgray]nspire the mind.[darkgray]'; }

[darkgray]// %d songs
[darkgray]// Ready to play
[darkgray]// Auto-play next enabled`, len(a.totalSongs))
	a.statusBar.SetText(welcomeMsg)

}

func (a *Application) renderSongTable() {
	for i := a.songTable.GetRowCount() - 1; i > 0; i-- {
		a.songTable.RemoveRow(i)
	}
	pageData := a.getCurrentPageData()
	
	for i, song := range pageData {
		row := i + 1
		
		// NCW风格的极简行设计
		var rowStyle tcell.Style
		rowStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault)
		
		// 序号列 - 右侧对齐的数字
		trackCell := tview.NewTableCell(fmt.Sprintf("%d:", row)).
			SetStyle(rowStyle.Foreground(tcell.ColorLightGreen)).
			SetAlign(tview.AlignRight)
		
		// 歌曲标题 - 主要内容
		titleCell := tview.NewTableCell(song.Title).
			SetStyle(rowStyle.Foreground(tcell.ColorWhite)).
			SetExpansion(1)
		
		// 艺术家
		artistCell := tview.NewTableCell(song.Artist).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetMaxWidth(25)
		
		// 专辑
		albumCell := tview.NewTableCell(song.Album).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetMaxWidth(25)
		
		// 时长
		durationCell := tview.NewTableCell(formatDuration(song.Duration)).
			SetStyle(rowStyle.Foreground(tcell.ColorGray)).
			SetAlign(tview.AlignRight)
		
		a.songTable.SetCell(row, 0, trackCell)
		a.songTable.SetCell(row, 1, titleCell)
		a.songTable.SetCell(row, 2, artistCell)
		a.songTable.SetCell(row, 3, albumCell)
		a.songTable.SetCell(row, 4, durationCell)
	}
	
	// 设置选中行的简约样式
	a.songTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))
	
	a.songTable.ScrollToBeginning()
}

func (a *Application) loadMusic() error {
	songs, err := a.subsonicClient.GetPlaylists()
	if err != nil {
		return fmt.Errorf("error get song list: %v", err)
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
				if e == nil {
					// 没有事件时短暂休眠以避免CPU占用过高
					time.Sleep(10 * time.Millisecond)
					continue
				}
				select {
				case c <- e:
				case <-ctx.Done():
					return
				}
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
		log.Printf("配置文件读取失败: %v", err)
		os.Exit(1)
	}

	for _, key := range required {
		if !viper.IsSet(key) {
			log.Printf("缺少必需的配置项: %s", key)
			os.Exit(1)
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
		log.Println("收到退出信号，正在清理资源...")
		
		// 停止MPV播放
		if app.mpvInstance != nil && app.mpvInstance.Mpv != nil {
			app.mpvInstance.Command([]string{"quit"})
			app.mpvInstance.TerminateDestroy()
		}
		
		cancel()
		app.application.Stop()
		
		// 强制退出
		go func() {
			time.Sleep(2 * time.Second)
			log.Println("强制退出")
			os.Exit(0)
		}()
	}()
	
	app.setupPagination()
	app.createHomepage()
	
	// 启动事件监听goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("事件监听goroutine异常退出: %v", r)
			}
		}()
		
		for {
			select {
			case event, ok := <-app.mpvInstance.EventChannel:
				if !ok {
					// 通道已关闭
					return
				}
				if event != nil && event.Event_Id == mpv.EVENT_END_FILE {
					// 歌曲结束，自动播放下一首
					app.application.QueueUpdateDraw(func() {
						app.playNextSong()
					})
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	
	go app.updateProgressBar()
	go func() {
		if err := app.loadMusic(); err != nil {
			app.application.QueueUpdateDraw(func() {
				app.statusBar.SetText("[red]加载失败: " + err.Error())
			})
		}
	}()

	// 运行应用程序
	log.Println("启动音乐播放器...")
	err = app.application.Run()
	
	// 程序退出时清理资源
	log.Println("程序正在退出，清理资源...")
	cancel() // 先取消context
	
	if app.mpvInstance != nil && app.mpvInstance.Mpv != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("清理MPV实例时出错: %v", r)
				}
			}()
			app.mpvInstance.Command([]string{"quit"})
			app.mpvInstance.TerminateDestroy()
		}()
	}
	
	if err != nil {
		log.Printf("应用程序运行错误: %v", err)
		os.Exit(1)
	}
	
	log.Println("程序正常退出")
	os.Exit(0)
}
