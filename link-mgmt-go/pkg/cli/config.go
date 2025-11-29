package cli

import (
	"fmt"
	"strings"

	"link-mgmt-go/pkg/config"

	"github.com/pelletier/go-toml/v2"
)

// ShowConfig displays the current configuration
func (a *App) ShowConfig() {
	data, err := toml.Marshal(a.cfg)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// SetConfig sets a configuration value
// Format: section.key=value (e.g., "database.url=postgres://...")
func (a *App) SetConfig(setStr string) error {
	parts := strings.SplitN(setStr, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected 'section.key=value'")
	}

	keyPath := strings.Split(parts[0], ".")
	value := parts[1]

	if len(keyPath) != 2 {
		return fmt.Errorf("invalid key format: expected 'section.key'")
	}

	section := keyPath[0]
	key := keyPath[1]

	switch section {
	case "database":
		switch key {
		case "url":
			a.cfg.Database.URL = value
		default:
			return fmt.Errorf("unknown database key: %s", key)
		}
	case "api":
		switch key {
		case "host":
			a.cfg.API.Host = value
		case "port":
			var port int
			if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
				return fmt.Errorf("invalid port value: %s", value)
			}
			a.cfg.API.Port = port
		default:
			return fmt.Errorf("unknown api key: %s", key)
		}
	case "cli":
		switch key {
		case "base_url":
			a.cfg.CLI.BaseURL = value
		case "api_key":
			a.cfg.CLI.APIKey = value
		case "scrape_timeout":
			var timeout int
			if _, err := fmt.Sscanf(value, "%d", &timeout); err != nil {
				return fmt.Errorf("invalid scrape_timeout value: %s", value)
			}
			a.cfg.CLI.ScrapeTimeout = timeout
		default:
			return fmt.Errorf("unknown cli key: %s", key)
		}
	default:
		return fmt.Errorf("unknown section: %s", section)
	}

	return config.Save(a.cfg)
}
