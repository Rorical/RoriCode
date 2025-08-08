package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Rorical/RoriCode/internal/app"
)

var rootCmd = &cobra.Command{
	Use:   "roricode",
	Short: "A Terminal User Interface chat application",
	Long:  `RoriCode is a TUI chat application built with Go and Bubble Tea, featuring OpenAI integration and profile management.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior: run the chat application
		application, err := app.NewApplication()
		if err != nil {
			log.Fatalf("Failed to create application: %v", err)
		}
		defer application.Stop()

		if err := application.Start(); err != nil {
			log.Fatalf("Application error: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Command execution error: %v", err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(profileCmd)
}