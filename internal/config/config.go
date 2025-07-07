package config

import (
	"fmt"
    "encoding/json"
    "os"
    "path/filepath"
	"errors"
)
const configFileName = ".gatorconfig.json"

type Config struct {
	DBUrl           string `json:"db_url"`
    CurrentUserName string `json:"current_user_name"`
}

type State struct {
	Config *Config
}

type Command struct {
	Name string
	Args []string
}

type CommandHandler func(*State, Command) error

type Commands struct {
	handlers map[string]CommandHandler
}

func NewCommands() *Commands {
	return &Commands{
		handlers: make(map[string]CommandHandler),
	}
}

func (c *Commands) Register(name string, handler CommandHandler) {
	c.handlers[name] = handler 
}

func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("Unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("username is required")
	}
	if len(cmd.Args) > 1 {
		return errors.New("login accepts only one argument (username)")
	}

	username := cmd.Args[0]
	s.Config.CurrentUserName = username

	if err := write(*s.Config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("User set to: %s\n", username)
	return nil
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