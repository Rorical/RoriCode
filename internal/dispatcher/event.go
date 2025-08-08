package dispatcher

import (
	"context"

	"github.com/Rorical/RoriCode/internal/eventbus"
)

// EventDispatcher handles routing events between core and UI
type EventDispatcher struct {
	eventBus *eventbus.EventBus
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewEventDispatcher(eventBus *eventbus.EventBus) *EventDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventDispatcher{
		eventBus: eventBus,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (ed *EventDispatcher) Start() {
	// No longer needed - UI handles events directly
}

func (ed *EventDispatcher) Stop() {
	ed.cancel()
}

func (ed *EventDispatcher) GetEventBus() *eventbus.EventBus {
	return ed.eventBus
}