package subsonic

import (
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	ClientID   string
	APIVersion string
	PageSize   int
	HttpClient *http.Client
}

type AlbumID3 struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	SongCount int   `json:"songCount"`
}

type Song struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Album        string    `json:"album"`
	Artist       string    `json:"artist"`
	Duration     int       `json:"duration"` // in seconds
	Track        int       `json:"track"`
	CoverArt     string    `json:"coverArt"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	Suffix       string    `json:"suffix"`
	BitRate      int       `json:"bitRate"`
	Path         string    `json:"path"`
	PlayCount    int       `json:"playCount"`
	Created      time.Time `json:"created"`
	AlbumID      string    `json:"albumId"`
	ArtistID     string    `json:"artistId"`
	IsVideo      bool      `json:"isVideo"`
	Played       time.Time `json:"played,omitempty"`
	ChannelCount int       `json:"channelCount"`
	SampleRate   int       `json:"samplingRate"`
}
