package domain

import (
	"sync"
	"time"
)

// Song represents a music track with metadata
type Song struct {
	ID           string
	Title        string
	Album        string
	Artist       string
	Duration     int // in seconds
	Track        int
	CoverArt     string
	Size         int64
	ContentType  string
	Suffix       string
	BitRate      int
	Path         string
	PlayCount    int
	Created      time.Time
	AlbumID      string
	ArtistID     string
	IsVideo      bool
	Played       *time.Time
	ChannelCount int
	SampleRate   int
}

// QueueItem represents an item in the playback queue
type QueueItem struct {
	ID       string
	URI      string
	Title    string
	Artist   string
	Duration int // in seconds
}

// PlayerState manages the current playback state in a thread-safe manner
type PlayerState struct {
	currentSong      *Song
	currentSongIndex int
	isPlaying        bool
	isLoading        bool
	mux              sync.RWMutex
}

// NewPlayerState creates a new PlayerState with default values
func NewPlayerState() *PlayerState {
	return &PlayerState{
		currentSongIndex: -1,
		isPlaying:        false,
		isLoading:        false,
	}
}

// GetState returns the current playback state (thread-safe)
func (s *PlayerState) GetState() (song *Song, index int, playing bool, loading bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.currentSong, s.currentSongIndex, s.isPlaying, s.isLoading
}

// SetLoading updates the loading state (thread-safe)
func (s *PlayerState) SetLoading(loading bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isLoading = loading
}

// SetPlaying updates the playing state (thread-safe)
func (s *PlayerState) SetPlaying(playing bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isPlaying = playing
}

// SetCurrentSong updates the current song and index (thread-safe)
func (s *PlayerState) SetCurrentSong(song *Song, index int) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.currentSong = song
	s.currentSongIndex = index
}
