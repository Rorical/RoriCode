package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Profile struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model"`
}

type Config struct {
	Profiles       map[string]Profile `json:"profiles"`
	ActiveProfile  string             `json:"active_profile"`
	currentProfile *Profile
}

func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Ensure config directory exists
	if err := ensureConfigDir(configPath); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing config or create default
	config, err := loadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate and set current profile
	if err := config.setCurrentProfile(); err != nil {
		return nil, fmt.Errorf("failed to set current profile: %w", err)
	}

	return config, nil
}

func (c *Config) IsValid() bool {
	return c.currentProfile != nil && c.currentProfile.APIKey != ""
}

func (c *Config) GetAPIKey() string {
	if c.currentProfile == nil {
		return ""
	}
	return c.currentProfile.APIKey
}

func (c *Config) GetModel() string {
	if c.currentProfile == nil {
		return "gpt-4o-mini"
	}
	return c.currentProfile.Model
}

func (c *Config) GetBaseURL() string {
	if c.currentProfile == nil {
		return ""
	}
	return c.currentProfile.BaseURL
}

func getConfigPath() (string, error) {
	var configDir string
	
	// Use RORICODE_HOME if set, otherwise use user's home directory
	if roriHome := os.Getenv("RORICODE_HOME"); roriHome != "" {
		configDir = roriHome
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = homeDir
	}
	
	return filepath.Join(configDir, ".roricode", "config.json"), nil
}

func ensureConfigDir(configPath string) error {
	configDir := filepath.Dir(configPath)
	return os.MkdirAll(configDir, 0755)
}

func loadConfigFile(configPath string) (*Config, error) {
	// If config file doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func createDefaultConfig(configPath string) (*Config, error) {
	config := &Config{
		Profiles: map[string]Profile{
			"default": {
				APIKey:  "",
				BaseURL: "",
				Model:   "gpt-4o-mini",
			},
		},
		ActiveProfile: "default",
	}

	// Save default config to file
	if err := saveConfig(config, configPath); err != nil {
		return nil, err
	}

	return config, nil
}

func saveConfig(config *Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	return saveConfig(c, configPath)
}

func (c *Config) setCurrentProfile() error {
	if c.Profiles == nil {
		return fmt.Errorf("no profiles defined")
	}

	profile, exists := c.Profiles[c.ActiveProfile]
	if !exists {
		// If active profile doesn't exist, try to use the first available profile
		for name, p := range c.Profiles {
			c.ActiveProfile = name
			profile = p
			exists = true
			break
		}
	}

	if !exists {
		return fmt.Errorf("no valid profiles found")
	}

	c.currentProfile = &profile
	return nil
}