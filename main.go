package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/yhkl-dev/NaviCLI/config"
	"github.com/yhkl-dev/NaviCLI/library"
	"github.com/yhkl-dev/NaviCLI/player"
	"github.com/yhkl-dev/NaviCLI/subsonic"
	"github.com/yhkl-dev/NaviCLI/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subsonicClient := subsonic.Init(
		cfg.Server.URL,
		cfg.Server.Username,
		cfg.Server.Password,
		cfg.Client.ID,
		cfg.Client.APIVersion,
		cfg.UI.PageSize,
		cfg.Player.GetHTTPTimeout(),
	)

	lib := library.NewSubsonicLibrary(subsonicClient)

	if err := lib.Ping(); err != nil {
		log.Fatalf("Can not connect to server %s, error: %v", cfg.Server.URL, err)
	}

	plr, err := player.NewMPVPlayer(ctx)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	app := ui.NewApp(ctx, cfg, lib, plr)

	// Logs default to stderr, which corrupts the terminal UI. Send them to a
	// file instead so the TUI stays clean. Best-effort: discard on failure.
	// Pre-Run fatals (config/ping/player) above still go to stderr.
	redirectLogs()

	var cleanupOnce sync.Once
	cleanup := func() {
		cancel()
		time.Sleep(10 * time.Millisecond)
		app.Stop()
		plr.Cleanup()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received exit signal, cleaning up resources...")
		cleanupOnce.Do(cleanup)

		go func() {
			time.Sleep(2 * time.Second)
			log.Println("Force quit.")
			os.Exit(0)
		}()
	}()

	log.Println("Starting NaviCLI...")
	if err := app.Run(); err != nil {
		log.Printf("Application error: %v", err)
		os.Exit(1)
	}

	log.Println("Program exiting, cleaning up...")
	cleanupOnce.Do(cleanup)

	log.Println("Program exit.")
	os.Exit(0)
}

// redirectLogs sends log output to a cache file so it does not corrupt the
// terminal UI while tview is running. Falls back to discarding logs if the
// file cannot be opened.
func redirectLogs() {
	dir, err := os.UserCacheDir()
	if err != nil {
		log.SetOutput(io.Discard)
		return
	}
	dir = filepath.Join(dir, "navicli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.SetOutput(io.Discard)
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "navicli.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.SetOutput(io.Discard)
		return
	}
	log.SetOutput(f)
}
