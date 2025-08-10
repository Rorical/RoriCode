package eventbus

import (
	"errors"
	"time"

	"github.com/Rorical/RoriCode/internal/models"
)

// UIEvent represents events sent from UI to Core
type UIEvent interface {
	UIEvent()
}

// CoreEvent represents events sent from Core to UI
type CoreEvent interface {
	CoreEvent()
}

// SendMessageEvent - UI requests core to send a message
type SendMessageEvent struct {
	Message string
}

func (e SendMessageEvent) UIEvent() {}

// StateUpdateEvent - Core pushes state changes to UI
type StateUpdateEvent struct {
	Messages     []models.Message
	IsProcessing bool
	Error        error
}

func (e StateUpdateEvent) CoreEvent() {}

// ConfirmationRequestEvent - Core requests user confirmation for dangerous operations
type ConfirmationRequestEvent struct {
	ID          string // Unique identifier for this confirmation request
	Operation   string // Description of the operation to confirm
	Command     string // The actual command/operation details
	Dangerous   bool   // Whether this is a potentially dangerous operation
}

func (e ConfirmationRequestEvent) CoreEvent() {}

// ConfirmationResponseEvent - UI sends user's confirmation decision back to Core
type ConfirmationResponseEvent struct {
	ID       string // Must match the ID from ConfirmationRequestEvent
	Approved bool   // User's decision: true = proceed, false = abort
}

func (e ConfirmationResponseEvent) UIEvent() {}

// EventBusError represents errors in event processing
type EventBusError struct {
	Operation string
	Err       error
	Timestamp time.Time
}

func (e EventBusError) Error() string {
	return e.Operation + ": " + e.Err.Error()
}

// CircuitBreakerState represents the state of circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state           CircuitBreakerState
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	if cb.state == CircuitOpen {
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
		}
	}
	return cb.state == CircuitOpen
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.failureCount = 0
	cb.state = CircuitClosed
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	if cb.failureCount >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

// EventBus handles communication between UI and Core with circuit breaker
type EventBus struct {
	uiToCore       chan UIEvent
	coreToUI       chan CoreEvent
	errorCallback  func(EventBusError)
	circuitBreaker *CircuitBreaker
}

func NewEventBus() *EventBus {
	return &EventBus{
		uiToCore:       make(chan UIEvent, 100),
		coreToUI:       make(chan CoreEvent, 100),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

func (eb *EventBus) SetErrorCallback(callback func(EventBusError)) {
	eb.errorCallback = callback
}

func (eb *EventBus) reportError(operation string, err error) {
	busError := EventBusError{
		Operation: operation,
		Err:       err,
		Timestamp: time.Now(),
	}
	
	eb.circuitBreaker.RecordFailure()
	
	if eb.errorCallback != nil {
		eb.errorCallback(busError)
	}
}

func (eb *EventBus) SendToCore(event UIEvent) error {
	if eb.circuitBreaker.IsOpen() {
		err := errors.New("circuit breaker is open")
		eb.reportError("SendToCore", err)
		return err
	}
	
	select {
	case eb.uiToCore <- event:
		eb.circuitBreaker.RecordSuccess()
		return nil
	default:
		err := errors.New("UI to Core channel is full")
		eb.reportError("SendToCore", err)
		return err
	}
}

func (eb *EventBus) SendToUI(event CoreEvent) error {
	if eb.circuitBreaker.IsOpen() {
		err := errors.New("circuit breaker is open")
		eb.reportError("SendToUI", err)
		return err
	}
	
	select {
	case eb.coreToUI <- event:
		eb.circuitBreaker.RecordSuccess()
		return nil
	default:
		err := errors.New("Core to UI channel is full")
		eb.reportError("SendToUI", err)
		return err
	}
}

func (eb *EventBus) UIToCore() <-chan UIEvent {
	return eb.uiToCore
}

func (eb *EventBus) CoreToUI() <-chan CoreEvent {
	return eb.coreToUI
}

func (eb *EventBus) GetCircuitBreakerState() CircuitBreakerState {
	return eb.circuitBreaker.state
}

func (eb *EventBus) Close() {
	close(eb.uiToCore)
	close(eb.coreToUI)
}