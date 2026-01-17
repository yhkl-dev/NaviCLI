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
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Subsonic client
	subsonicClient := subsonic.Init(
		cfg.Server.URL,
		cfg.Server.Username,
		cfg.Server.Password,
		cfg.Client.ID,
		cfg.Client.APIVersion,
		cfg.UI.PageSize,
		cfg.Player.GetHTTPTimeout(),
	)

	// Initialize library
	lib := library.NewSubsonicLibrary(subsonicClient)

	// Initialize player
	plr, err := player.NewMPVPlayer(ctx)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	// Initialize UI
	app := ui.NewApp(ctx, cfg, lib, plr)

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received exit signal, cleaning up resources...")

		// Cancel context first to stop event loops and background goroutines
		cancel()
		// Give time for context cancellation to propagate
		time.Sleep(10 * time.Millisecond)
		// Then stop UI
		app.Stop()
		// Finally cleanup player
		plr.Cleanup()

		go func() {
			time.Sleep(2 * time.Second)
			log.Println("Force quit.")
			os.Exit(0)
		}()
	}()

	// Run the application
	log.Println("Starting NaviCLI...")
	if err := app.Run(); err != nil {
		log.Printf("Application error: %v", err)
		os.Exit(1)
	}

	// Cleanup
	log.Println("Program exiting, cleaning up...")
	// Cancel context first to stop background goroutines
	cancel()
	// Give time for context cancellation to propagate
	time.Sleep(10 * time.Millisecond)
	// Cleanup player (must come after context cancellation)
	plr.Cleanup()

	log.Println("Program exit.")
	os.Exit(0)
}
