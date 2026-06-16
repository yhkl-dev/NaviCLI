package library

import "github.com/yhkl-dev/NaviCLI/domain"

type Library interface {
	GetRandomSongs(count int) ([]domain.Song, error)
	GetAlbumSongs(albumType string) ([]domain.Song, error)
	SearchSongs(query string, limit int) ([]domain.Song, error)
	GetPlayURL(songID string) string
	GetCoverArtURL(coverArtID string) string
	Ping() error
}
