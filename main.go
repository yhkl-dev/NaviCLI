package main

import (
	"fmt"
	"log"

	"github.com/yhkl-dev/NaviCLI/subsonic"
)

func main() {
	client := subsonic.Init(
		"http://192.168.2.5:49153",
		"yhkl",
		"young331",
		"goplayer",
		"1.16.1",
	)
	songs, err := client.GetPlaylists()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(songs)

	if err != nil {
		log.Fatal("搜索失败:", err)
	}

	if len(songs) == 0 {
		log.Fatal("未找到歌曲")
	}

	fmt.Println(songs)
	song := songs[0]
	fmt.Printf("正在播放: %s - %s\n", song.Title, song.Album)

	playURL := client.GetPlayURL(song.ID, 192)
	fmt.Println("GetPlayURL", playURL)
	if err := subsonic.PlayStream(playURL); err != nil {
		log.Fatal("播放失败:", err)
	}

	fmt.Println("按Enter键停止播放...")
	fmt.Scanln()
}
