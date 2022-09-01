package test

import (
	"context"
	"log"

	"github.com/colevoss/awesome-go-realtime/server"
)

type Test struct {
}

func (t *Test) Start(factory *server.ChannelFactory) {
	log.Printf("test: starting test service")

	factory.BeforeJoin(t.TestBeforeJoin)
	factory.Join(t.TestJoin)
	factory.Leave(t.Leave)
	factory.Handle("test", t.TestHandler)
	factory.Handle("other-test", t.OtherTestHandler)
}

type TestData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Age  int32  `json:"age"`
}

func (t *Test) TestHandler(ctx context.Context, event *server.Event) error {
	var testData TestData

	event.Data(&testData)

	log.Printf("Test event for chann: %s: %v", event.Channel.Name, event.Param("id"))
	log.Printf("My Test %#v", testData)

	res := &TestData{
		Id:   event.Conn.Id.String(),
		Name: "RESPONSE MESSAGE",
		Age:  30,
	}

	event.Broadcast("res-event", res)

	return nil
}

func (t *Test) TestBeforeJoin(ctx context.Context, event *server.Event) error {
	log.Printf("test: %s", event.Conn)

	return nil
}

func (t *Test) TestJoin(ctx context.Context, event *server.Event) error {
	log.Printf("test: Yeah! I joined the channel id: %s", event.Param("id"))

	event.Ack(map[string]interface{}{
		"id":     event.Conn.Id,
		"status": "joined",
	})

	event.Broadcast("joined", map[string]interface{}{
		"id": event.Conn.Id,
	})

	return nil
}

func (t *Test) Leave(ctx context.Context, event *server.Event) error {
	log.Printf("test: Conn leaving %s", event.Conn)

	event.Broadcast("person-left", map[string]interface{}{
		"id":            event.Conn.Id,
		"what-happened": "they left",
	})

	return nil
}

func (t *Test) OtherTestHandler(ctx context.Context, event *server.Event) error {
	var testData TestData

	event.Data(&testData)

	log.Printf("OTHER: %s: %v", event.Channel.Name, event.Param("id"))
	log.Printf("My Test %#v", testData)

	return nil
}
