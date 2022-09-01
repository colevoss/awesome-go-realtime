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
	Hub      *Hub
	mu       sync.Mutex
	upgrader *websocket.Upgrader
}

func NewRealtimeServer() *RealtimeServer {
	return &RealtimeServer{
		Hub: newHub(),
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool {
				return true
			},
		},
	}
}

func (s *RealtimeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("[rts] upgrade:", err)
		return
	}

	log.Printf("[rts] Creating connection")

	ctx := r.Context()
	conn := newConnection(ctx, c, s)

	defer conn.closeConnection()

	log.Printf("[rts] %s created", conn)

	conn.start()
}

type ChannelHandler interface {
	Start(*ChannelFactory)
}

func (s *RealtimeServer) RegisterChannel(path string, ch ChannelHandler) {
	factory := NewChannelFactory(path)

	ch.Start(factory)
	s.Hub.registerChannelFactory(factory)
}

func (s *RealtimeServer) RegisterChannelFactory(ch *ChannelFactory) {
	s.Hub.registerChannelFactory(ch)
}
