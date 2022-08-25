package server

import (
	"fmt"
	"log"
	"sync"
)

const (
	BeforeJoin  = "__CHANNEL_BEFORE_JOIN__"
	Join        = "__CHANNEL_JOIN__"
	BeforeLeave = "__CHANNEL_BEFORE_LEAVE__"
	Leave       = "__CHANNEL_LEAVE__"
)

type internalChannel struct {
	mu sync.RWMutex

	Name   string
	Path   string
	Params *Params

	factory     *ChannelFactory
	connections ConnectionMap
	hub         *Hub
	register    chan *Context
	unregister  chan *Context
	event       chan *Context

	done chan bool
}

func (c *internalChannel) String() string {
	return fmt.Sprintf("chan:%s", c.Name)
}

func newChannel(name string, params *Params, factory *ChannelFactory, hub *Hub) *internalChannel {
	return &internalChannel{
		Name:        name,
		Path:        factory.path,
		Params:      params,
		factory:     factory,
		connections: make(ConnectionMap),
		hub:         hub,

		register:   make(chan *Context),
		unregister: make(chan *Context),

		event: make(chan *Context),

		done: make(chan bool, 1),
	}
}

func (c *internalChannel) start() {
	log.Printf("[%s] Starting channel", c)
	defer c.closeChannel()

	for {
		select {
		case ctx := <-c.register:
			if err := c.handleRegister(ctx); err != nil {
				log.Printf("[%s] Error during channel register %v", c, err)
			}

		case ctx := <-c.unregister:
			c.removeConnection(ctx)

		case ctx := <-c.event:
			c.handleEvent(ctx.Event(), ctx)

		case <-c.done:
			return
		}
	}
}

func (c *internalChannel) sendMessage(msg *ServerMessage) {
	bytes, err := msg.Marshal()

	if err != nil {
		log.Printf("[%s] Error marshalling message %v", c, err)
		return
	}

	for conn := range c.connections {
		select {
		case conn.byteSend <- bytes:
		default:
		}
	}
}

func (c *internalChannel) broadcastMessage(msg *ServerMessage, conn *Connection) {
	bytes, err := msg.Marshal()

	if err != nil {
		log.Printf("[%s] Error marshalling message %v", c, err)
		return
	}

	for connection := range c.connections {
		if connection == conn {
			continue
		}

		log.Printf("[%s] Sending msg to %s", c, conn)
		select {
		case connection.byteSend <- bytes:
		default:
		}
	}
}

func (c *internalChannel) handleEvent(event string, ctx *Context) error {
	handler, exists := c.factory.handlers[event]

	if !exists {
		log.Printf("[%s] Handler '%s' not available for channel %s", c, event, c.Name)
		return nil
	}

	log.Printf("[%s] Handling event: %s", c, event)

	return handler.handler(ctx)
}

func (c *internalChannel) handleRegister(ctx *Context) error {
	err := c.handleEvent(BeforeJoin, ctx)

	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.connections[ctx.Conn] = true
	ctx.Conn.addChannel(c)

	return c.handleEvent(Join, ctx)
}

func (c *internalChannel) removeConnection(ctx *Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.connections[ctx.Conn]

	if !exists {
		return
	}

	log.Printf("[%s] Removing connection %s", c, ctx.Conn)
	delete(c.connections, ctx.Conn)

	err := c.handleEvent(Leave, ctx)

	if err != nil {
		log.Println(err)
	}

	ctx.Conn.removeChannel(c)

	if len(c.connections) == 0 {
		c.done <- true
	}
}

func (c *internalChannel) closeChannel() {
	log.Printf("[%s] Closing channel", c)
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hub.closeChannel(c)

	close(c.register)
	close(c.event)
	close(c.unregister)
	close(c.done)
}

/**
Sends message to all connected clients includeing sender
*/
func (c *internalChannel) Emit(event string, data interface{}) {
	serverMessage := c.newServerMessage(event, data)
	c.sendMessage(serverMessage)
}

/**
Sends message to all clients excluding sender
*/
func (c *internalChannel) Broadcast(event string, data interface{}, conn *Connection) {
	serverMessage := c.newServerMessage(event, data)
	c.broadcastMessage(serverMessage, conn)
}

func (c *internalChannel) newServerMessage(event string, data interface{}) *ServerMessage {
	return &ServerMessage{
		Message: Message{
			Channel: c.Name,
			Type:    ServerEvent,
		},
		Event: event,
		Data:  data,
	}
}

type ChannelFactory struct {
	mu sync.Mutex
	*DotPath
	handlers map[string]channelEntry
}

func NewChannelFactory(path string) *ChannelFactory {
	return &ChannelFactory{
		DotPath:  NewDotPath(path),
		handlers: map[string]channelEntry{},
	}
}

func (cf *ChannelFactory) newChannel(name string, params *Params, hub *Hub) *internalChannel {
	return newChannel(name, params, cf, hub)
}

type ChannelEventHandler func(*Context) error

type channelEntry struct {
	handler ChannelEventHandler
	event   string
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
