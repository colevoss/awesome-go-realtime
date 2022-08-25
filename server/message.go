package server

import "encoding/json"

type ConnectionEvent string

const (
	Subscribe   ConnectionEvent = "Subscribe"
	Unsubscribe                 = "Unsubscribe"
	ClientEvent                 = "ClientEvent"
	ServerEvent                 = "ServerEvent"
)

type Message struct {
	Type    ConnectionEvent `json:"type"`
	Channel string          `json:"channel"`
}

type ConnectionMessage struct {
	Message
	Event   string          `json:"event"`
	RawData json.RawMessage `json:"data"`
}

type SubscribeMessage = ConnectionMessage
type UnsubscribeMessage = ConnectionMessage

func (cm *ConnectionMessage) Data(v interface{}) error {
	return json.Unmarshal(cm.RawData, v)
}

type ServerMessage struct {
	Message
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

func (sm *ServerMessage) Marshal() ([]byte, error) {
	return json.Marshal(sm)
}
