package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var newLine = []byte{'\n'}

type Connection struct {
	conn     *websocket.Conn
	mu       sync.RWMutex
	Id       uuid.UUID
	server   *RealtimeServer
	byteSend chan []byte
	close    chan bool
	channels map[*internalChannel]bool
}

func (c *Connection) String() string {
	return fmt.Sprintf("conn:%s", c.Id)
}

func newConnection(conn *websocket.Conn, server *RealtimeServer) *Connection {
	return &Connection{
		Id:       uuid.New(),
		conn:     conn,
		server:   server,
		byteSend: make(chan []byte),
		channels: make(map[*internalChannel]bool),
	}
}

func (c *Connection) read() {
	log.Printf("[%s] starting read", c)

	defer func() {
		log.Printf("[%s] Closing conn read", c)
		c.closeConnection()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(c.pongHandler)

	for {
		msg := &ConnectionMessage{}
		err := c.conn.ReadJSON(msg)

		if err != nil {
			log.Printf("[%s] error %v", c, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[%s] error %v", c, err)
			}
			break
		}

		c.handleMessage(msg, c)
	}
}

func (c *Connection) write() {
	log.Printf("[%s] Starting write", c)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Printf("[%s] Closing writer", c)
		ticker.Stop()
	}()

	for {
		select {
		case byteMessage, ok := <-c.byteSend:
			// ok used to determine that connection is closed
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeBytes(byteMessage); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			log.Printf("[%s] Ping sent", c)

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Connection) pongHandler(x string) error {
	log.Printf("[%s] Pong received", c)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	return nil
}

func (c *Connection) writeBytes(msg []byte) error {
	w, err := c.conn.NextWriter(websocket.TextMessage)

	if err != nil {
		return err
	}

	w.Write(msg)

	if err = w.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Connection) closeConnection() {
	log.Printf("[%s] Closing connection", c)

	c.conn.Close()

	for channel := range c.channels {
		ctx := &Context{
			Conn:    c,
			Channel: channel,
		}
		channel.unregister <- ctx
	}

	c.server.unregister <- c
	close(c.byteSend)
}

func (c *Connection) addChannel(channel *internalChannel) {
	log.Printf("[%s] Adding channel %s to connection", c, channel.Name)
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channels[channel] = true
}

func (c *Connection) removeChannel(channel *internalChannel) {
	log.Printf("[%s] Removing channel %s from connection", c, channel.Name)
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.channels[channel]; ok {
		delete(c.channels, channel)
		log.Printf("[%s] Channel %s removed from connection", c, channel.Name)
	}
}

func (c *Connection) writeMessage(message *ServerMessage) (err error) {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))

	err = c.conn.WriteJSON(message)
	return
}

func (c *Connection) createContext(msg *ConnectionMessage, conn *Connection) (ctx *Context, ok bool) {
	var channel *internalChannel

	switch msg.Type {
	case Subscribe:
		channel, ok = c.server.Hub.findOrOpenChannel(msg.Channel)
	case Unsubscribe, ClientEvent:
		channel, ok = c.server.Hub.findChannel(msg.Channel)
	}

	if !ok {
		log.Printf("WTF %v %#v", ok, channel)
		return
	}

	ctx = NewContext(channel, conn, msg)

	return
}

func (c *Connection) handleMessage(msg *ConnectionMessage, conn *Connection) {
	ctx, ok := c.createContext(msg, conn)

	if !ok {
		return
	}

	switch ctx.Msg.Type {
	case Subscribe:
		ctx.Channel.register <- ctx

	case Unsubscribe:
		ctx.Channel.unregister <- ctx

	case ClientEvent:
		ctx.Channel.event <- ctx
	}
}
