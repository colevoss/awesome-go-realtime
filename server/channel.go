package server

import (
	"context"
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

type Channel struct {
	mu sync.RWMutex

	Name   string
	Path   string
	Params *Params

	factory     *ChannelFactory
	connections ConnectionMap
	hub         *Hub
}

func (c *Channel) String() string {
	return fmt.Sprintf("chan:%s", c.Name)
}

/**
Sends message to all connected clients includeing sender
*/
func (c *Channel) Emit(event string, data interface{}) {
	serverMessage := c.newServerMessage(event, data)
	c.sendMessage(serverMessage)
}

/**
Sends message to all clients excluding sender
*/
func (c *Channel) Broadcast(event string, data interface{}, conn *Connection) {
	serverMessage := c.newServerMessage(event, data)
	c.broadcastMessage(serverMessage, conn)
}

func newChannel(name string, params *Params, factory *ChannelFactory, hub *Hub) *Channel {
	return &Channel{
		Name:        name,
		Path:        factory.path,
		Params:      params,
		factory:     factory,
		connections: make(ConnectionMap),
		hub:         hub,
	}
}

func (c *Channel) handleEvent(ctx context.Context, event *Event) {
	switch event.Msg.Type {
	case Subscribe:
		c.handleRegister(ctx, event)
	case Unsubscribe:
		c.removeConnection(ctx, event)
	case ClientEvent:
		c.handleClientEvent(ctx, event.Name(), event)
	}
}

func (c *Channel) sendMessage(msg *ServerMessage) {
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

func (c *Channel) broadcastMessage(msg *ServerMessage, conn *Connection) {
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

func (c *Channel) handleClientEvent(ctx context.Context, eventName string, event *Event) error {
	handler, exists := c.factory.handler(eventName)

	if !exists {
		log.Printf("[%s] Handler '%s' not available for channel %s", c, eventName, c.Name)
		err := NewServerError("Handler not available for channel", ServerErrorFields{
			"channel": c.Name,
			"event":   eventName,
		})
		event.Conn.handleError(err)

		return nil
	}

	log.Printf("[%s] Handling event: %s", c, eventName)

	return handler.handler(ctx, event)
}

func (c *Channel) handleBuiltinEvent(ctx context.Context, eventName string, event *Event) error {
	handler, exists := c.factory.handler(eventName)

	if !exists {
		return nil
	}

	return handler.handler(ctx, event)
}

func (c *Channel) handleRegister(ctx context.Context, event *Event) error {
	err := c.handleBuiltinEvent(ctx, BeforeJoin, event)

	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.connections[event.Conn] = true
	event.Conn.addChannel(c)

	return c.handleBuiltinEvent(ctx, Join, event)
}

func (c *Channel) removeConnection(ctx context.Context, event *Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.connections[event.Conn]

	if !exists {
		return
	}

	log.Printf("[%s] Removing connection %s", c, event.Conn)
	delete(c.connections, event.Conn)

	err := c.handleBuiltinEvent(ctx, Leave, event)

	if err != nil {
		log.Println(err)
	}

	event.Conn.removeChannel(c)

	if len(c.connections) == 0 {
		c.closeChannel()
	}
}

func (c *Channel) closeChannel() {
	log.Printf("[%s] Closing channel", c)

	c.hub.closeChannel(c)
}

func (c *Channel) newServerMessage(event string, data interface{}) *ServerMessage {
	return &ServerMessage{
		Message: Message{
			Channel: c.Name,
			Type:    ServerEvent,
		},
		Event: event,
		Data:  data,
	}
}
