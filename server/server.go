package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ConnectionMap map[*Connection]bool

type IRealTimeServer interface {
	Serve(http.ResponseWriter, *http.Request)
}

type RealtimeServer struct {
	Hub *Hub

	mu          sync.Mutex
	connections ConnectionMap
	register    chan *Connection
	unregister  chan *Connection
	upgrader    *websocket.Upgrader
}

func NewRealtimeServer() *RealtimeServer {
	return &RealtimeServer{
		Hub:         NewHub(),
		connections: make(ConnectionMap),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		upgrader:    &websocket.Upgrader{},
	}
}

func (s *RealtimeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("[rts] upgrade:", err)
		return
	}

	log.Printf("[rts] Creating connection")

	conn := newConnection(c, s)

	log.Printf("[rts] %s created", conn)

	s.register <- conn

	go conn.read()
	go conn.write()
}

func (s *RealtimeServer) Start() {
	for {
		select {
		case connection := <-s.register:
			s.registerConnection(connection)

		case connection := <-s.unregister:
			s.removeConnection(connection)
		}
	}
}

type ChannelHandler interface {
	Start(*ChannelFactory)
}

func (s *RealtimeServer) RegisterChannel(path string, ch ChannelHandler) {
	factory := NewChannelFactory(path)

	ch.Start(factory)
	s.Hub.RegisterChannelFactory(factory)
}

func (s *RealtimeServer) registerConnection(conn *Connection) {
	log.Printf("[rts] Registering connection %s", conn)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections[conn] = true

	log.Printf("[rts] %s registered", conn)
}

func (s *RealtimeServer) removeConnection(conn *Connection) {
	log.Printf("[rts] Removing connection %s", conn)
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.connections[conn]

	if !ok {
		log.Printf("[rts] Conn not found. Cannot remove")
		return
	}

	delete(s.connections, conn)
	log.Printf("[rts] Conn removed %s", conn)
}
