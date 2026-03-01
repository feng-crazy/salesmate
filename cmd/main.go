package main

import (
	"log"
	"os"

	"salesmate/cmd/commands"
)

func main() {
	// Set up basic logging
	log.SetOutput(os.Stdout)

	// Check if any arguments are provided
	if len(os.Args) == 1 {
		// If no args, show help
		commands.Execute()
		return
	}

	// Otherwise, execute the command
	commands.Execute()
}
