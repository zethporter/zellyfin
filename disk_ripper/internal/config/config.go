package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	TMDB   TMDBConfig   `toml:"tmdb"`
	Drive  DriveConfig  `toml:"drive"`
	Output OutputConfig `toml:"output"`
}

type TMDBConfig struct {
	APIKey string `toml:"api_key"`
}

type DriveConfig struct {
	Device string `toml:"device"`
}

type OutputConfig struct {
	Dir     string `toml:"dir"`
	TempDir string `toml:"temp_dir"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to load config: %w", err)
	}

	if key := os.Getenv("TMDB_API_KEY"); key != "" {
		cfg.TMDB.APIKey = key
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
