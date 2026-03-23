package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	VaultPath   string            `mapstructure:"vault_path"`
	Model       string            `mapstructure:"model"`
	Meilisearch MeilisearchConfig `mapstructure:"meilisearch"`
}

type MeilisearchConfig struct {
	Host   string `mapstructure:"host"`
	APIKey string `mapstructure:"api_key"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}

	viper.SetConfigName(".ngram")
	viper.SetConfigType("yml")
	viper.AddConfigPath(home)

	viper.SetDefault("model", "cloud")
	viper.SetDefault("meilisearch.host", "http://127.0.0.1:7700")
	viper.SetDefault("meilisearch.api_key", "")

	viper.SetEnvPrefix("NGRAM")
	viper.BindEnv("vault_path")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.VaultPath == "" {
		return nil, fmt.Errorf("vault_path not set (configure in ~/.ngram.yml or set NGRAM_VAULT_PATH)")
	}

	cfg.VaultPath = expandHome(cfg.VaultPath, home)

	info, err := os.Stat(cfg.VaultPath)
	if err != nil {
		return nil, fmt.Errorf("vault path %q: %w", cfg.VaultPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault path %q is not a directory", cfg.VaultPath)
	}

	return &cfg, nil
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if len(path) > 1 && path[:2] == "~/" {
		return filepath.Join(home, path[2:])
	}
	return path
}
