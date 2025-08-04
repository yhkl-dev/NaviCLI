package subsonic

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

func playWithTempFile(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	tmpFile, err := os.CreateTemp("", "audio-*.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()
	cmd := exec.Command("afplay", tmpFile.Name())
	cmd.Run()
}

func PlayStream(url string) error {
	switch runtime.GOOS {
	case "darwin":
		// cmd = exec.Command("afplay", url)
		playWithTempFile(url)
	case "linux":
		panic("not implemented")
	case "windows":
		panic("not implemented")
	default:
		return fmt.Errorf("unsupported platform")
	}
	return nil
}
