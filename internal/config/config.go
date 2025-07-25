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
	"strconv"

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
		return fmt.Errorf("'%s' requires %d argument(s)", cmdName, expected)
	case len(args) > expected:
		return fmt.Errorf("'%s' accepts only %d argument(s)", cmdName, expected)
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

func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if err := validateArgs(cmd.Args, 2, "addfeed"); err != nil {
		return err
	}

	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:			uuid.New(),
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
		Name:		cmd.Args[0],
		Url:		cmd.Args[1],
		UserID:		user.ID,
	})

	fmt.Printf("Created feed: %+v\n", feed)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("url '%s' already exists", cmd.Args[1])
		}
		return fmt.Errorf("database error: %w", err)
	}

	feedFollow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:			uuid.New(),
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
		UserID:		user.ID,
		FeedID:		feed.ID,
	})

	fmt.Println("NEW feedFollow:", feedFollow)

	return nil
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
    return func(s *State, cmd Command) error {
        user, err := getUser(s.DB, s.Config.CurrentUserName)
        if err != nil {
            return fmt.Errorf("failed to get user: %w", err)
        }

        return handler(s, cmd, *user)
    }
}

func HandlerFeeds(s *State, cmd Command) error {
	feeds, err := s.DB.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Get Feeds failed: %w", err)
	}

	for i, _ := range feeds {
		fmt.Println("Information about feed number - ", i)
		fmt.Println("Feed's name:", feeds[i].Name)
		fmt.Println("Feed's url:", feeds[i].Url)
		fmt.Println("Feed's username:", feeds[i].Username)
		fmt.Println()
	}
	return nil
}

func HandlerFollow(s *State, cmd Command, user database.User) error {
	if err := validateArgs(cmd.Args, 1, "follow"); err != nil {
		return err
	}

	url := cmd.Args[0]

	currentFeed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed wth next reason: %w", err)
	}


	feed, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:			uuid.New(),
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
		UserID:		user.ID,
		FeedID:		currentFeed.ID,
	})

	fmt.Println("NEW feed:", feed)

	return nil
}


func HandlerFollowing(s *State, cmd Command, user database.User) error {
	if err := validateArgs(cmd.Args, 0, "following"); err != nil {
		return err
	}

	FeedFollowsForUser, err := s.DB.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed wth next reason: %w", err)
	}

	if len(FeedFollowsForUser) == 0 {
		fmt.Println("The current user is NOT subscribed to any feeds yet")
	} else {
		fmt.Println("The names of the feeds the current user is following:")

		for i, _ := range FeedFollowsForUser {
			fmt.Println("*", FeedFollowsForUser[i].FeedName)
		}
	}

	return nil
}


func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	if err := validateArgs(cmd.Args, 1, "unfollow"); err != nil {
		return err
	}

	if err := s.DB.DeleteFeedFollowByURL(context.Background(), database.DeleteFeedFollowByURLParams{
		UserID:	user.ID,
		Url:	cmd.Args[0],
	}); err != nil {
		return fmt.Errorf("failed to unfollow: %w", err)
	}

	fmt.Printf("Unfollowed feed with URL: %s\n", cmd.Args[0])

	return nil
}

func HandlerAgg(s *State, cmd Command) error {
	if err := validateArgs(cmd.Args, 1, "agg"); err != nil {
		return err
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
        return fmt.Errorf("invalid duration format: %w", err)
    }
	fmt.Println("Collecting feeds every ", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}

	return nil
}

func scrapeFeeds(s *State) {
	// 1. Получить следующий фид для обработки
	feed, err := s.DB.GetNextFeedToFetch(context.Background())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("No feeds to fetch")
			return
		}
		fmt.Printf("Error getting next feed: %v\n", err)
		return
	}

	fmt.Printf("\nFetching feed: %s (%s)\n", feed.Name, feed.Url)

	// 2. Получить и обработать фид
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("Error fetching feed %s: %v\n", feed.Url, err)
		return
	}

	// 3. Вывести элементы
	for _, item := range rssFeed.Channel.Item {
		// fmt.Printf("%d. %s\n", i+1, item.Title)
		// Парсим дату публикации (с обработкой разных форматов)
        publishedAt, err := parseFeedDate(item.PubDate)
        if err != nil {
            fmt.Printf("Error parsing date '%s': %v\n", item.PubDate, err)
            publishedAt = time.Now() // Используем текущее время как fallback
        }

		// Создаем параметры для сохранения поста
		_, err = s.DB.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: sql.NullTime{Time:  publishedAt, Valid: !publishedAt.IsZero()},
			FeedID:      feed.ID,
		})

		// Обрабатываем ошибки
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				// Пост уже существует - пропускаем
				continue
			}
			fmt.Printf("Error saving post '%s': %v\n", item.Title, err)
		} else {
			fmt.Printf("Saved post: %s\n", item.Title)
		}
	}

	// 4. Обновить время последнего фетчинга
	err = s.DB.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Printf("Error marking feed as fetched: %v\n", err)
	}
}

func parseFeedDate(dateStr string) (time.Time, error) {
    formats := []string{
        time.RFC1123,
        time.RFC1123Z,
        time.RFC822,
        time.RFC822Z,
        time.RFC3339,
        "Mon, 2 Jan 2006 15:04:05 -0700",
        "2006-01-02T15:04:05Z",
    }

    for _, format := range formats {
        t, err := time.Parse(format, dateStr)
        if err == nil {
            return t, nil
        }
    }
    
    return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}

func HandlerBrowse(s *State, cmd Command, user database.User) error {
	limitPost := int32(2);

	if len(cmd.Args) == 1 {
        limit, err := strconv.ParseInt(cmd.Args[0], 10, 32)
        if err != nil {
            return fmt.Errorf("invalid limit value: %w", err)
        }
        limitPost = int32(limit)
	}
	
	posts, err := s.DB.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID:	user.ID,
		Limit:	limitPost,
	})

	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	for i, post := range posts {
		fmt.Printf("\n=== Post %d ===\n", i+1)
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)

		// Decode HTML entities in text fields
		plainDesc := html.UnescapeString(post.Description.String)
		fmt.Printf("Description:\n%s\n", plainDesc)

		fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04"))
		fmt.Printf("Feed: %s\n", post.FeedName)
		fmt.Println("------------------------")
	}

	return nil
}