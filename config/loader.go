package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath(".")

	defaults := DefaultConfig()
	viper.SetDefault("ui.page_size", defaults.UI.PageSize)
	viper.SetDefault("ui.progress_bar_width", defaults.UI.ProgressBarWidth)
	viper.SetDefault("ui.max_column_width", defaults.UI.MaxColumnWidth)
	viper.SetDefault("player.http_timeout", defaults.Player.HTTPTimeout)
	viper.SetDefault("client.id", defaults.Client.ID)
	viper.SetDefault("client.api_version", defaults.Client.APIVersion)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	required := []string{
		"server.url",
		"server.username",
		"server.password",
	}
	for _, key := range required {
		if !viper.IsSet(key) {
			return nil, fmt.Errorf("missing required config: %s", key)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
