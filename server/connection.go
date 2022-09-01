package server

import (
	"context"
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
	Id uuid.UUID

	conn *websocket.Conn

	mu       sync.RWMutex
	channels map[*Channel]bool
	server   *RealtimeServer

	ctx  context.Context
	stop func()

	byteSend chan []byte

	pingTicker *time.Ticker

	stopper sync.Once
}

type connIdKey = string

var ConnIdKey = connIdKey("connId")

func (c *Connection) String() string {
	return fmt.Sprintf("conn:%s", c.Id)
}

func newConnection(ctx context.Context, conn *websocket.Conn, server *RealtimeServer) *Connection {
	c := &Connection{
		Id:         uuid.New(),
		conn:       conn,
		server:     server,
		byteSend:   make(chan []byte),
		channels:   make(map[*Channel]bool),
		pingTicker: time.NewTicker(pingPeriod),
	}

	ctx, stop := context.WithCancel(context.WithValue(ctx, ConnIdKey, c.Id))

	c.ctx = ctx
	c.stop = stop

	return c
}

func (c *Connection) ConnId(ctx context.Context) any {
	return ctx.Value(ConnIdKey)
}

func (c *Connection) start() {
	c.read()
}

func (c *Connection) read() {
	log.Printf("[%s] starting read", c)

	go c.write()

	defer func() {
		log.Printf("[%s] Closing conn read", c)
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// c.conn.SetPingHandler(func(string) error {
	// 	log.Printf("[%s] Ping received", c)
	// 	if err := c.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
	// 		return err
	// 	}
	//
	// 	return nil
	// })

	c.conn.SetPongHandler(func(string) error {
		log.Printf("[%s] Pong received", c)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msg := &ClientMessage{}
		err := c.conn.ReadJSON(msg)

		if err != nil {
			log.Printf("[%s] error %v!??????????????", c, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[%s] error %v!!!!!!!!!!!!", c, err)
			}

			log.Printf("[%s] WHAT IS HAPPENING???????", c)
			break
		}

		go c.handleMessage(c.ctx, msg)
	}
}

func (c *Connection) write() {
	log.Printf("[%s] Starting write", c)
	defer func() {
		log.Printf("[%s] Closing writer", c)
		c.closeConnection()
	}()

	for {
		select {
		case byteMessage := <-c.byteSend:
			if err := c.writeBytes(byteMessage); err != nil {
				return
			}

		case <-c.pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			log.Printf("[%s] Ping sent", c)

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[%s] Ping message error %s", c, err)
				return
			}

		case <-c.ctx.Done():
			log.Printf("DONE NOW YEAH!!!!!!!!!!! %v", c.ctx.Err())
			return
		}
	}
}

func (c *Connection) closeConnection() {
	log.Printf("[%s] Closing connection", c)

	c.stopper.Do(func() {
		c.stop()
		c.pingTicker.Stop()
		c.conn.Close()

		for channel := range c.channels {
			ctx := &Event{
				Conn:    c,
				Channel: channel,
			}
			go channel.removeConnection(c.ctx, ctx)
		}

		close(c.byteSend)
	})
}

func (c *Connection) writeBytes(msg []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))

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

func (c *Connection) addChannel(channel *Channel) {
	log.Printf("[%s] Adding channel %s to connection", c, channel.Name)
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channels[channel] = true
}

func (c *Connection) removeChannel(channel *Channel) {
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

func (c *Connection) handleMessage(ctx context.Context, msg *ClientMessage) {
	var channel *Channel
	var ok bool

	switch msg.Type {
	case Subscribe:
		channel, ok = c.server.Hub.findOrOpenChannel(msg.Channel)
	case Unsubscribe, ClientEvent:
		channel, ok = c.server.Hub.findChannel(msg.Channel)
	}

	if !ok {
		c.handleError(NewServerError("Channel not found", ServerErrorFields{
			"channel": msg.Channel,
		}))
		return
	}

	event := NewEvent(channel, c, msg)
	event.Channel.handleEvent(ctx, event)
}

func (c *Connection) handleError(err error) {
	var serverErr *ServerError
	switch err.(type) {
	case *ServerError:
		serverErr = err.(*ServerError)
	default:
		serverErr = NewServerError(err.Error(), ServerErrorFields{})
	}

	bytes, _ := serverErr.Marshal()
	c.byteSend <- bytes
}
