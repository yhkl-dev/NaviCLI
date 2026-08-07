package library

import (
	"fmt"
	"log"
	"time"

	"github.com/yhkl-dev/NaviCLI/domain"
	"github.com/yhkl-dev/NaviCLI/subsonic"
)

type SubsonicLibrary struct {
	client *subsonic.Client
}

func NewSubsonicLibrary(client *subsonic.Client) *SubsonicLibrary {
	return &SubsonicLibrary{
		client: client,
	}
}

// GetAllSongs fetches every song in the library: it paginates the album list,
// then fetches each album's tracks concurrently (8 workers) so a few hundred
// albums do not become a few hundred sequential requests.
func (s *SubsonicLibrary) GetAllSongs() ([]domain.Song, error) {
	const (
		batchSize   = 50
		concurrency = 8
	)

	type albumRef struct{ id, name string }
	var albums []albumRef
	offset := 0
	for {
		batch, err := s.client.GetAlbumList2("alphabeticalByName", batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("getAlbumList2: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		for _, a := range batch {
			albums = append(albums, albumRef{id: a.ID, name: a.Name})
		}
		if len(batch) < batchSize {
			break
		}
		offset += batchSize
	}

	jobs := make(chan albumRef, len(albums))
	for _, a := range albums {
		jobs <- a
	}
	close(jobs)

	type result struct {
		songs []domain.Song
		err   error
	}
	results := make(chan result, len(albums))
	for i := 0; i < concurrency; i++ {
		go func() {
			for a := range jobs {
				songs, err := s.client.GetAlbum(a.id)
				if err != nil {
					results <- result{err: fmt.Errorf("album %s: %w", a.name, err)}
					continue
				}
				results <- result{songs: convertToDomainSongs(songs)}
			}
		}()
	}

	allSongs := make([]domain.Song, 0, len(albums)*10)
	for i := 0; i < len(albums); i++ {
		r := <-results
		if r.err != nil {
			log.Printf("skip album: %v", r.err)
			continue
		}
		allSongs = append(allSongs, r.songs...)
	}
	return allSongs, nil
}

func (s *SubsonicLibrary) SearchSongs(query string, limit int) ([]domain.Song, error) {
	songs, err := s.client.SearchSongs(query, limit)
	if err != nil {
		return nil, err
	}
	return convertToDomainSongs(songs), nil
}

func (s *SubsonicLibrary) GetPlayURL(songID string) string {
	return s.client.GetPlayURL(songID)
}

func (s *SubsonicLibrary) GetCoverArtURL(coverArtID string) string {
	return s.client.GetCoverArtURL(coverArtID)
}

func (s *SubsonicLibrary) Ping() error {
	return s.client.GetServerInfo()
}

func convertToDomainSongs(songs []subsonic.Song) []domain.Song {
	domainSongs := make([]domain.Song, len(songs))
	for i, song := range songs {
		domainSongs[i] = convertToDomainSong(song)
	}
	return domainSongs
}

func convertToDomainSong(song subsonic.Song) domain.Song {
	var played *time.Time
	if !song.Played.IsZero() {
		played = &song.Played
	}

	return domain.Song{
		ID:           song.ID,
		Title:        song.Title,
		Album:        song.Album,
		Artist:       song.Artist,
		Duration:     song.Duration,
		Track:        song.Track,
		CoverArt:     song.CoverArt,
		Size:         song.Size,
		ContentType:  song.ContentType,
		Suffix:       song.Suffix,
		BitRate:      song.BitRate,
		Path:         song.Path,
		PlayCount:    song.PlayCount,
		Created:      song.Created,
		AlbumID:      song.AlbumID,
		ArtistID:     song.ArtistID,
		IsVideo:      song.IsVideo,
		Played:       played,
		ChannelCount: song.ChannelCount,
		SampleRate:   song.SampleRate,
	}
}
