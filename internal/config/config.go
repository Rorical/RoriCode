package config

import (
	"os"
)

type Config struct {
	OpenAIAPIKey string
	OpenAIModel  string
	OpenAIBaseURL string
}

func LoadConfig() *Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_KEY")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_API_BASE")
	}

	return &Config{
		OpenAIAPIKey:  apiKey,
		OpenAIModel:   model,
		OpenAIBaseURL: baseURL,
	}
}

func (c *Config) IsValid() bool {
	return c.OpenAIAPIKey != ""
}