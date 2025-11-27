package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	// Database
	Database struct {
		URL string `toml:"url"`
	} `toml:"database"`

	// API
	API struct {
		Port int    `toml:"port"`
		Host string `toml:"host"`
	} `toml:"api"`

	// CLI
	CLI struct {
		APIBaseURL string `toml:"api_base_url"`
		APIKey     string `toml:"api_key"`
	} `toml:"cli"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Database.URL = "postgres://localhost/linkmgmt?sslmode=disable"
	cfg.API.Port = 8080
	cfg.API.Host = "0.0.0.0"
	cfg.CLI.APIBaseURL = "http://localhost:8080"
	cfg.CLI.APIKey = ""
	return cfg
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "link-mgmt")
	return filepath.Join(configDir, "config.toml"), nil
}

// Load reads configuration from ~/.config/link-mgmt/config.toml
// Creates the file with defaults if it doesn't exist
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// Expand ~ in path if needed
	if strings.HasPrefix(configPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = strings.Replace(configPath, "~", homeDir, 1)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create default config file
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults for any missing values
	defaultCfg := DefaultConfig()
	if cfg.Database.URL == "" {
		cfg.Database.URL = defaultCfg.Database.URL
	}
	if cfg.API.Port == 0 {
		cfg.API.Port = defaultCfg.API.Port
	}
	if cfg.API.Host == "" {
		cfg.API.Host = defaultCfg.API.Host
	}
	if cfg.CLI.APIBaseURL == "" {
		cfg.CLI.APIBaseURL = defaultCfg.CLI.APIBaseURL
	}

	return &cfg, nil
}

// Save writes the configuration to the config file
func Save(cfg *Config) error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Expand ~ in path if needed
	if strings.HasPrefix(configPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = strings.Replace(configPath, "~", homeDir, 1)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to TOML
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
