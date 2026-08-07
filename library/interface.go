package library

import "github.com/yhkl-dev/NaviCLI/domain"

type Library interface {
	GetAllSongs() ([]domain.Song, error)
	SearchSongs(query string, limit int) ([]domain.Song, error)
	GetPlayURL(songID string) string
	GetCoverArtURL(coverArtID string) string
	Ping() error
}
