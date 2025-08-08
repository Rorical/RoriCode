package cmd

import (
	"fmt"
	"log"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/Rorical/RoriCode/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage API profiles",
	Long:  `Manage API profiles for different providers and configurations.`,
}

var listProfilesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		fmt.Printf("Active Profile: %s\n\n", cfg.ActiveProfile)
		fmt.Println("Available Profiles:")
		for name, profile := range cfg.Profiles {
			marker := ""
			if name == cfg.ActiveProfile {
				marker = " (active)"
			}
			fmt.Printf("  %s%s\n", name, marker)
			fmt.Printf("    Model: %s\n", profile.Model)
			if profile.BaseURL != "" {
				fmt.Printf("    Base URL: %s\n", profile.BaseURL)
			}
			hasKey := "No"
			if profile.APIKey != "" {
				hasKey = "Yes"
			}
			fmt.Printf("    API Key: %s\n", hasKey)
			fmt.Println()
		}
	},
}

var showProfileCmd = &cobra.Command{
	Use:   "show [profile-name]",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		profileName := args[0]
		profile, exists := cfg.Profiles[profileName]
		if !exists {
			log.Fatalf("Profile '%s' does not exist", profileName)
		}

		fmt.Printf("Profile: %s\n", profileName)
		fmt.Printf("Model: %s\n", profile.Model)
		fmt.Printf("Base URL: %s\n", profile.BaseURL)
		hasKey := "Not set"
		if profile.APIKey != "" {
			hasKey = "Set (hidden for security)"
		}
		fmt.Printf("API Key: %s\n", hasKey)
	},
}

var addProfileCmd = &cobra.Command{
	Use:   "add [profile-name]",
	Short: "Add a new profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			prompt := promptui.Prompt{
				Label: "Profile name",
			}
			profileName, err = prompt.Run()
			if err != nil {
				log.Fatalf("Prompt failed: %v", err)
			}
		}

		if _, exists := cfg.Profiles[profileName]; exists {
			log.Fatalf("Profile '%s' already exists", profileName)
		}

		profile := config.Profile{}

		// Prompt for API Key
		apiKeyPrompt := promptui.Prompt{
			Label: "API Key",
			Mask:  '*',
		}
		profile.APIKey, err = apiKeyPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}

		// Prompt for Model
		modelPrompt := promptui.Prompt{
			Label:   "Model",
			Default: "gpt-4o-mini",
		}
		profile.Model, err = modelPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}

		// Prompt for Base URL (optional)
		baseURLPrompt := promptui.Prompt{
			Label: "Base URL (optional)",
		}
		profile.BaseURL, err = baseURLPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}

		// Add profile to config
		cfg.Profiles[profileName] = profile

		// Save config
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		fmt.Printf("Profile '%s' added successfully!\n", profileName)
	},
}

var editProfileCmd = &cobra.Command{
	Use:   "edit [profile-name]",
	Short: "Edit an existing profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			// Let user select from existing profiles
			profileNames := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				profileNames = append(profileNames, name)
			}

			if len(profileNames) == 0 {
				log.Fatalf("No profiles available to edit")
			}

			prompt := promptui.Select{
				Label: "Select profile to edit",
				Items: profileNames,
			}
			_, profileName, err = prompt.Run()
			if err != nil {
				log.Fatalf("Selection failed: %v", err)
			}
		}

		profile, exists := cfg.Profiles[profileName]
		if !exists {
			log.Fatalf("Profile '%s' does not exist", profileName)
		}

		// Edit API Key
		apiKeyPrompt := promptui.Prompt{
			Label:   "API Key",
			Default: profile.APIKey,
			Mask:    '*',
		}
		newAPIKey, err := apiKeyPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}
		profile.APIKey = newAPIKey

		// Edit Model
		modelPrompt := promptui.Prompt{
			Label:   "Model",
			Default: profile.Model,
		}
		newModel, err := modelPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}
		profile.Model = newModel

		// Edit Base URL
		baseURLPrompt := promptui.Prompt{
			Label:   "Base URL",
			Default: profile.BaseURL,
		}
		newBaseURL, err := baseURLPrompt.Run()
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}
		profile.BaseURL = newBaseURL

		// Update profile in config
		cfg.Profiles[profileName] = profile

		// Save config
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		fmt.Printf("Profile '%s' updated successfully!\n", profileName)
	},
}

var deleteProfileCmd = &cobra.Command{
	Use:   "delete [profile-name]",
	Short: "Delete a profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			// Let user select from existing profiles
			profileNames := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				profileNames = append(profileNames, name)
			}

			if len(profileNames) == 0 {
				log.Fatalf("No profiles available to delete")
			}

			prompt := promptui.Select{
				Label: "Select profile to delete",
				Items: profileNames,
			}
			_, profileName, err = prompt.Run()
			if err != nil {
				log.Fatalf("Selection failed: %v", err)
			}
		}

		if _, exists := cfg.Profiles[profileName]; !exists {
			log.Fatalf("Profile '%s' does not exist", profileName)
		}

		// Confirm deletion
		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Delete profile '%s'? (y/N)", profileName),
			IsConfirm: true,
		}
		_, err = confirmPrompt.Run()
		if err != nil {
			fmt.Println("Deletion cancelled")
			return
		}

		// Check if we're deleting the active profile
		if cfg.ActiveProfile == profileName {
			// Find another profile to make active
			for name := range cfg.Profiles {
				if name != profileName {
					cfg.ActiveProfile = name
					break
				}
			}
			// If this was the last profile, create a new default one
			if len(cfg.Profiles) == 1 {
				cfg.ActiveProfile = "default"
				cfg.Profiles["default"] = config.Profile{
					APIKey:  "",
					BaseURL: "",
					Model:   "gpt-4o-mini",
				}
			}
		}

		// Delete the profile
		delete(cfg.Profiles, profileName)

		// Save config
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		fmt.Printf("Profile '%s' deleted successfully!\n", profileName)
	},
}

var switchProfileCmd = &cobra.Command{
	Use:   "switch [profile-name]",
	Short: "Switch to a different profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			// Let user select from existing profiles
			profileNames := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				if name != cfg.ActiveProfile {
					profileNames = append(profileNames, name)
				}
			}

			if len(profileNames) == 0 {
				fmt.Println("No other profiles available to switch to")
				return
			}

			prompt := promptui.Select{
				Label: "Select profile to switch to",
				Items: profileNames,
			}
			_, profileName, err = prompt.Run()
			if err != nil {
				log.Fatalf("Selection failed: %v", err)
			}
		}

		if _, exists := cfg.Profiles[profileName]; !exists {
			log.Fatalf("Profile '%s' does not exist", profileName)
		}

		cfg.ActiveProfile = profileName

		// Save config
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

		fmt.Printf("Switched to profile '%s'\n", profileName)
	},
}

func init() {
	// Add subcommands to profile
	profileCmd.AddCommand(listProfilesCmd)
	profileCmd.AddCommand(showProfileCmd)
	profileCmd.AddCommand(addProfileCmd)
	profileCmd.AddCommand(editProfileCmd)
	profileCmd.AddCommand(deleteProfileCmd)
	profileCmd.AddCommand(switchProfileCmd)
}