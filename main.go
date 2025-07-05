package main

import (
	"fmt"
	"log"
	"github.com/BabichevDima/aggregator/internal/config"
)

func main() {
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	// Set the current user to your name (replace "dimababichau" with your actual name)
	if err := cfg.SetUser("dimababichau"); err != nil {
		log.Fatalf("Error updating config: %v", err)
	}

	// Read the config file again
	updatedCfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading updated config: %v", err)
	}

	// Print the contents of the config struct
	fmt.Println(updatedCfg)
	fmt.Printf("%+v\n", *updatedCfg)
}