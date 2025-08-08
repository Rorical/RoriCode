package main

import (
	"fmt"
	"os"

	"github.com/Rorical/RoriCode/internal/app"
)

func main() {
	// Create and configure application
	application, err := app.NewApplication()
	if err != nil {
		fmt.Printf("Failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Start application
	if err := application.Start(); err != nil {
		fmt.Printf("Application error: %v\n", err)
		os.Exit(1)
	}

	// Clean shutdown
	application.Stop()
}