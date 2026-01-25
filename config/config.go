package config

import "time"

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	UI     UIConfig     `mapstructure:"ui"`
	Player PlayerConfig `mapstructure:"player"`
	Client ClientConfig `mapstructure:"client"`
}

type ServerConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type UIConfig struct {
	PageSize         int `mapstructure:"page_size"`
	ProgressBarWidth int `mapstructure:"progress_bar_width"`
	MaxColumnWidth   int `mapstructure:"max_column_width"`
}

type PlayerConfig struct {
	HTTPTimeout int `mapstructure:"http_timeout"`
}

type ClientConfig struct {
	ID         string `mapstructure:"id"`
	APIVersion string `mapstructure:"api_version"`
}

func (p *PlayerConfig) GetHTTPTimeout() time.Duration {
	return time.Duration(p.HTTPTimeout) * time.Second
}

func (c *Config) Validate() error {
	return nil
}

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
