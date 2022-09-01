package server

import "encoding/json"

type ConnectionEvent string

const (
	Subscribe              ConnectionEvent = "Subscribe"
	Unsubscribe                            = "Unsubscribe"
	ClientEvent                            = "ClientEvent"
	ServerEvent                            = "ServerEvent"
	ServerErrorMessageType                 = "ServerError"
)

type Message struct {
	Type    ConnectionEvent `json:"type"`
	Channel string          `json:"channel"`
}

type ClientMessage struct {
	Message
	Event   string          `json:"event"`
	RawData json.RawMessage `json:"data"`
}

type SubscribeMessage = ClientMessage
type UnsubscribeMessage = ClientMessage

func (cm *ClientMessage) Data(v interface{}) error {
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

type ServerError struct {
	Type ConnectionEvent `json:"type"`
	Msg  string          `json:"error"`
	Data interface{}     `json:"data"`
}

func (se *ServerError) Error() string {
	return se.Msg
}

func (se *ServerError) Marshal() ([]byte, error) {
	return json.Marshal(se)
}

type ServerErrorFields = map[string]interface{}

func NewServerError(error string, data ServerErrorFields) *ServerError {
	return &ServerError{
		Type: ServerErrorMessageType,
		Msg:  error,
		Data: data,
	}
}
