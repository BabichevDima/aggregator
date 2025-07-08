package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"io"
	"path/filepath"
	"strings"
	"time"
	"html"

	"net/http"

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

// TODO: RSS
type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}
// TODO: RSS

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

    s.Config.CurrentUserName = username
	if err := updateConfigUser(s.Config, username); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

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

func HandlerUsers(s *State, cmd Command) error {
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Get Users failed: %w", err)
	}

	for i, _ := range users {
		message := "* " + users[i]
		if s.Config.CurrentUserName == users[i] {
			message += " (current)"
		}
		fmt.Println(message)
	}
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

func HandlerAgg(s *State, cmd Command) error {
	feedURL := "https://www.wagslane.dev/index.xml"

	rssFeed, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Fetch Feed failed: %w", err)
	}

	fmt.Println(rssFeed)
	fmt.Println("Title:", rssFeed.Channel.Item[0].Title)
	fmt.Println("Description:", rssFeed.Channel.Item[0].Description)

	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// set request headers
	req.Header.Set("User-Agent", "gator")

	// create a new client and make the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer res.Body.Close()

    // Check status code
    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
    }

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var rssFeed RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}

	// Decode HTML entities in text fields
	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
    rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)

    for i := range rssFeed.Channel.Item {
        item := &rssFeed.Channel.Item[i]
        item.Title = html.UnescapeString(item.Title)
        item.Description = html.UnescapeString(item.Description)
    }

	return &rssFeed, nil
}