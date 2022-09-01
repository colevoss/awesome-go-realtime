package server

import (
	"context"
	"sync"
)

type ChannelFactory struct {
	*DotPath

	mu       sync.Mutex
	handlers map[string]channelEntry
}

type ChannelEventHandler func(context.Context, *Event) error

type channelEntry struct {
	handler ChannelEventHandler
	event   string
}

func NewChannelFactory(path string) *ChannelFactory {
	return &ChannelFactory{
		DotPath:  NewDotPath(path),
		handlers: make(map[string]channelEntry),
	}
}

func (cf *ChannelFactory) Handle(event string, handler ChannelEventHandler) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if handler == nil {
		panic("cf: nil handler")
	}

	if event == "" {
		panic("cf: invalid event")
	}

	if _, exists := cf.handlers[event]; exists {
		panic("cf: multiple handlers for " + event)
	}

	entry := channelEntry{event: event, handler: handler}

	cf.handlers[event] = entry
}

func (cf *ChannelFactory) BeforeJoin(handler ChannelEventHandler) {
	cf.Handle(BeforeJoin, handler)
}

func (cf *ChannelFactory) Join(handler ChannelEventHandler) {
	cf.Handle(Join, handler)
}

func (cf *ChannelFactory) BeforeLeave(handler ChannelEventHandler) {
	cf.Handle(BeforeLeave, handler)
}

func (cf *ChannelFactory) Leave(handler ChannelEventHandler) {
	cf.Handle(Leave, handler)
}

func (cf *ChannelFactory) newChannel(name string, params *Params, hub *Hub) *Channel {
	return newChannel(name, params, cf, hub)
}

func (cf *ChannelFactory) handler(handlerName string) (channelEntry, bool) {
	h, exists := cf.handlers[handlerName]
	return h, exists
}
