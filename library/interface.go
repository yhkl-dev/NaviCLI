package library

import "github.com/yhkl-dev/NaviCLI/domain"

// Library defines the interface for music library operations
// This abstraction allows NaviCLI to work with different backends (Subsonic, Spotify, local files, etc.)
type Library interface {
	// GetRandomSongs retrieves a specified number of random songs from the library
	GetRandomSongs(count int) ([]domain.Song, error)

	// SearchSongs searches for songs matching the query
	// Returns up to 'limit' results
	SearchSongs(query string, limit int) ([]domain.Song, error)

	// GetPlayURL returns the streaming URL for a given song ID
	GetPlayURL(songID string) string

	// GetCoverArtURL returns the URL for album cover art
	GetCoverArtURL(coverArtID string) string

	// Ping verifies connectivity to the music server
	Ping() error
}
