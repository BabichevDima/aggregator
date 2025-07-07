package config

import (
	"fmt"
    "encoding/json"
    "os"
    "path/filepath"
	"errors"
	"github.com/BabichevDima/aggregator/internal/database"
    "github.com/google/uuid"
    "time"
    "context"
	"strings"
	
	"database/sql"
)
const configFileName = ".gatorconfig.json"

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

    // Проверка существования пользователя
    _, err := s.DB.GetUser(context.Background(), username)
    if err != nil {
        if err == sql.ErrNoRows {
			return fmt.Errorf("user '%s' does not exist\n", username)
        }
        fmt.Printf("Error checking user: %v\n", err)
        os.Exit(1)
    }

	s.Config.CurrentUserName = username

	if err := write(*s.Config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("User set to: %s\n", username)
	return nil
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("username is required")
	}
	if len(cmd.Args) > 1 {
		return errors.New("Register accepts only one argument (username)")
	}

	username := cmd.Args[0]
    // Генерация UUID и времени
    id := uuid.New()
    now := time.Now()

	_, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
        ID:        id,
        CreatedAt: now,
        UpdatedAt: now,
        Name:      username,
    })
	// Обработка ошибок (включая случай существующего пользователя)
    if err != nil {
        if strings.Contains(err.Error(), "duplicate key") {
            return fmt.Errorf("user '%s' already exists", username)
        }
        return fmt.Errorf("failed to create user: %w", err)
    }

	// Обновление текущего пользователя в конфиге
    s.Config.CurrentUserName = username
    if err := write(*s.Config); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    // Вывод результата
    fmt.Printf("User '%s' created successfully\n", username)
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