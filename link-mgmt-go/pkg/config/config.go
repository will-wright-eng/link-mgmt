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
		BaseURL       string `toml:"base_url"` // Base URL for all services (via nginx)
		APIKey        string `toml:"api_key"`
		ScrapeTimeout int    `toml:"scrape_timeout"` // Timeout for scraping operations in seconds
	} `toml:"cli"`

	// Scraper
	Scraper struct {
		BaseURL string `toml:"base_url"` // Base URL for scraper service
	} `toml:"scraper"`
}

// DefaultConfig returns a config with default values
// Database defaults match docker-compose.yml settings
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Database.URL = "postgres://link_mgmt_user:link_mgmt_pwd@localhost:5432/link_mgmt_db?sslmode=disable"
	cfg.API.Port = 8080
	cfg.API.Host = "0.0.0.0"
	cfg.CLI.BaseURL = "http://localhost" // nginx reverse proxy on port 80
	cfg.CLI.APIKey = ""
	cfg.CLI.ScrapeTimeout = 30               // 30 seconds default
	cfg.Scraper.BaseURL = "http://localhost" // scraper service default
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

		// Override with environment variables if set (useful for Docker)
		if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
			cfg.Database.URL = dbURL
		}
		if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
			cfg.CLI.BaseURL = baseURL
		}

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
	if cfg.CLI.ScrapeTimeout == 0 {
		cfg.CLI.ScrapeTimeout = defaultCfg.CLI.ScrapeTimeout
	}
	if cfg.CLI.BaseURL == "" {
		cfg.CLI.BaseURL = defaultCfg.CLI.BaseURL
	}
	if cfg.Scraper.BaseURL == "" {
		cfg.Scraper.BaseURL = defaultCfg.Scraper.BaseURL
	}

	// Override with environment variables if set (useful for Docker)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
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
