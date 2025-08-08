package models

// AppModel represents the UI state - only local UI concerns
type AppModel struct {
	Messages         []Message // Current messages to display
	Input            string    // User input field
	Status           string    // Status bar text
	Loading          bool      // Loading state from core
	LoadingDots      int       // Animation counter for loading dots
	Width            int       // Terminal width
	Height           int       // Terminal height
	ChatServiceReady bool      // Whether chat service is available
}