package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
    "os"
    "path/filepath"
	"strings"
    "time"

	"github.com/BabichevDima/aggregator/internal/database"
	"github.com/google/uuid"
)
const (
	configFileName = ".gatorconfig.json"
	configFilePerm = 0644
)

type Config struct {
	DBUrl           string `json:"db_url"`
    CurrentUserName string `json:"current_user_name"`
}

type State struct {
	Config *Config
	DB  *database.Queries
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

	if _, err := getUser(s.DB, username); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := updateConfigUser(s.Config, username); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	fmt.Printf("User set to: %s\n", username)
	return nil
}

func getUser(db *database.Queries, username string) (*database.User, error) {
	user, err := db.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user '%s' does not exist", username)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

func updateConfigUser(cfg *Config, username string) error {
	cfg.CurrentUserName = username
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func saveConfig(cfg *Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

func HandlerRegister(s *State, cmd Command) error {
	if err := validateArgs(cmd.Args, 1, "register"); err != nil {
		return err
	}

	username := cmd.Args[0]
    if err := createUser(s.DB, username); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Обновление текущего пользователя в конфиге
    s.Config.CurrentUserName = username
	if err := updateConfigUser(s.Config, username); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

    // Вывод результата
    fmt.Printf("User '%s' created successfully\n", username)
	return nil
}

func HandlerReset(s *State, cmd Command) error {
    if err := s.DB.DeleteAllUsers(context.Background()); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

    fmt.Printf("Table was successful reset to a blank state\n")
	return nil
}

func validateArgs(args []string, expected int, cmdName string) error {
	switch {
	case len(args) < expected:
		return fmt.Errorf("%s requires %d argument(s)", cmdName, expected)
	case len(args) > expected:
		return fmt.Errorf("%s accepts only %d argument(s)", cmdName, expected)
	}
	return nil
}

func createUser(db *database.Queries, username string) error {
	_, err := db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("user '%s' already exists", username)
		}
		return fmt.Errorf("database error: %w", err)
	}

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

	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}