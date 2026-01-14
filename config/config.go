package config

import "time"

// Config represents the complete application configuration
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	UI     UIConfig     `mapstructure:"ui"`
	Player PlayerConfig `mapstructure:"player"`
	Client ClientConfig `mapstructure:"client"`
}

// ServerConfig contains Navidrome server connection settings
type ServerConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// UIConfig contains user interface settings
type UIConfig struct {
	PageSize         int `mapstructure:"page_size"`
	ProgressBarWidth int `mapstructure:"progress_bar_width"`
	MaxColumnWidth   int `mapstructure:"max_column_width"`
}

// PlayerConfig contains playback and HTTP client settings
type PlayerConfig struct {
	HTTPTimeout int `mapstructure:"http_timeout"` // in seconds
}

// ClientConfig contains Subsonic API client settings
type ClientConfig struct {
	ID         string `mapstructure:"id"`
	APIVersion string `mapstructure:"api_version"`
}

// GetHTTPTimeout returns the HTTP timeout as a time.Duration
func (p *PlayerConfig) GetHTTPTimeout() time.Duration {
	return time.Duration(p.HTTPTimeout) * time.Second
}

// Validate checks if all required configuration values are set
func (c *Config) Validate() error {
	// Server validation is handled by viper's IsSet checks in main
	return nil
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		UI: UIConfig{
			PageSize:         20,
			ProgressBarWidth: 30,
			MaxColumnWidth:   40,
		},
		Player: PlayerConfig{
			HTTPTimeout: 30,
		},
		Client: ClientConfig{
			ID:         "navicli",
			APIVersion: "1.16.1",
		},
	}
}
