package main

import (
	// PostgreSQL driver (imported for side effects)
	_ "github.com/lib/pq"
	"log"
	"github.com/BabichevDima/aggregator/internal/config"
	"github.com/BabichevDima/aggregator/internal/database"
	"os"
	"database/sql"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Error: not enough arguments were provided!")
	}
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
	defer db.Close()

    // Создание экземпляра queries
    dbQueries := database.New(db)

	state := &config.State{
		DB:		dbQueries,
		Config:	cfg,
	}

	commands := config.NewCommands()
	commands.Register("login", config.HandlerLogin)
	commands.Register("register", config.HandlerRegister)
	commands.Register("reset", config.HandlerReset)
	commands.Register("users", config.HandlerUsers)

	cmdName := os.Args[1]
	var cmdArgs []string
	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}

	cmd := config.Command{
		Name: cmdName,
		Args: cmdArgs,
	}

	if err := commands.Run(state, cmd); err != nil {
		log.Fatalf("Error: %v", err)
	}
}