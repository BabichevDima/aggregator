package main

import (
	"log"
	"github.com/BabichevDima/aggregator/internal/config"
    "os"
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

	state := &config.State{
		Config: cfg,
	}

	commands := config.NewCommands()
	commands.Register("login", config.HandlerLogin)

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