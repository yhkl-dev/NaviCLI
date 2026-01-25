package main

import (
	"context"
	"log"
	"os"
	"os/signal"
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received exit signal, cleaning up resources...")

		cancel()
		time.Sleep(10 * time.Millisecond)
		app.Stop()
		plr.Cleanup()

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
	cancel()
	time.Sleep(10 * time.Millisecond)
	plr.Cleanup()

	log.Println("Program exit.")
	os.Exit(0)
}
