package config

import (
	"fmt"
    "encoding/json"
    "os"
    "path/filepath"
)
const configFileName = ".gatorconfig.json"

type Config struct {
	DBUrl           string `json:"db_url"`
    CurrentUserName string `json:"current_user_name"`
}

func Read() (*Config, error) {
	path, err := getConfigFilePath()

	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// SetUser updates the current user and saves the config
func (c *Config) SetUser(newName string) error {
	c.CurrentUserName = newName
	return write(*c)
}

// getConfigFilePath returns the full path to the config file
func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

// write saves the config to disk
func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}