package models

// ConfirmationRequest represents a confirmation request (avoiding import cycle)
type ConfirmationRequest struct {
	ID          string // Unique identifier for this confirmation request
	Operation   string // Description of the operation to confirm
	Command     string // The actual command/operation details
	Dangerous   bool   // Whether this is a potentially dangerous operation
}

// AppModel represents the UI state - only local UI concerns
type AppModel struct {
	Messages            []Message            // Current messages to display
	Input               string               // User input field
	Status              string               // Status bar text
	Loading             bool                 // Loading state from core
	LoadingDots         int                  // Animation counter for loading dots
	Width               int                  // Terminal width
	Height              int                  // Terminal height
	ChatServiceReady    bool                 // Whether chat service is available
	PendingConfirmation *ConfirmationRequest // Current confirmation request
}