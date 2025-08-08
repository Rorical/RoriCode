package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/Rorical/RoriCode/internal/app"
	"github.com/Rorical/RoriCode/internal/config"
)

var useCmd = &cobra.Command{
	Use:   "use [profile-name]",
	Short: "Switch to a profile and start the chat app",
	Long:  `Switch to the specified profile and immediately start the chat application.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profileName := args[0]

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		// Check if profile exists
		if _, exists := cfg.Profiles[profileName]; !exists {
			log.Fatalf("Profile '%s' does not exist", profileName)
		}

		// Switch to the profile
		cfg.ActiveProfile = profileName

		// Save config with new active profile
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		// Start the chat application
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

func init() {
	rootCmd.AddCommand(useCmd)
}