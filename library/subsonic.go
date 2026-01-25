package library

import (
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

func (s *SubsonicLibrary) GetRandomSongs(count int) ([]domain.Song, error) {
	songs, err := s.client.GetRandomSongs(count)
	if err != nil {
		return nil, err
	}
	return convertToDomainSongs(songs), nil
}

func (s *SubsonicLibrary) SearchSongs(query string, limit int) ([]domain.Song, error) {
	songs, err := s.client.SearchSongs(query)
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
