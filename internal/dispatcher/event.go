package dispatcher

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/update"
)

// EventDispatcher handles routing events between core and UI
type EventDispatcher struct {
	eventBus    *eventbus.EventBus
	uiEventChan chan update.CoreEventMsg
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewEventDispatcher(eventBus *eventbus.EventBus) *EventDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventDispatcher{
		eventBus:    eventBus,
		uiEventChan: make(chan update.CoreEventMsg, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (ed *EventDispatcher) Start() {
	go ed.listenForCoreEvents()
}

func (ed *EventDispatcher) Stop() {
	ed.cancel()
	close(ed.uiEventChan)
}

func (ed *EventDispatcher) GetUIEventChannel() <-chan update.CoreEventMsg {
	return ed.uiEventChan
}

func (ed *EventDispatcher) GetEventBus() *eventbus.EventBus {
	return ed.eventBus
}

func (ed *EventDispatcher) listenForCoreEvents() {
	for {
		select {
		case <-ed.ctx.Done():
			return
		case event, ok := <-ed.eventBus.CoreToUI():
			if !ok {
				return
			}
			// Non-blocking send to UI
			select {
			case ed.uiEventChan <- update.CoreEventMsg{Event: event}:
			case <-ed.ctx.Done():
				return
			default:
				// Log dropped event instead of silently dropping
				// TODO: Add proper logging
			}
		}
	}
}

func (ed *EventDispatcher) ListenForUIEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-ed.uiEventChan:
			return msg
		case <-ed.ctx.Done():
			return nil
		}
	}
}