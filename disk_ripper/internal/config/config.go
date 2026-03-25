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
	SFTP   SFTPConfig   `toml:"sftp"`
}

type TMDBConfig struct {
	APIKey string `toml:"api_key"`
}

type DriveConfig struct {
	Device string `toml:"device"`
}

type OutputConfig struct {
	Dir string `toml:"dir"`
}

type SFTPConfig struct {
	Host       string `toml:"host"`
	Port       string `toml:"port"`
	User       string `toml:"user"`
	RemotePath string `toml:"remote_path"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to load config: %w", err)
	}

	if key := os.Getenv("TMDB_API_KEY"); key != "" {
		cfg.TMDB.APIKey = key
	}
	if host := os.Getenv("SFTP_HOST"); host != "" {
		cfg.SFTP.Host = host
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
