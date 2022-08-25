package test

import (
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

func (t *Test) TestHandler(ctx *server.Context) error {
	var testData TestData

	ctx.Data(&testData)

	log.Printf("Test event for chann: %s: %v", ctx.Channel.Name, ctx.Params("id"))
	log.Printf("My Test %#v", testData)

	res := &TestData{
		Id:   "RESPONSE",
		Name: "RESPONSE MESSAGE",
		Age:  30,
	}

	ctx.Broadcast("res-event", res)

	return nil
}

func (t *Test) TestBeforeJoin(ctx *server.Context) error {
	log.Print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	log.Print("!!!!!!!! Before Join Called !!!!!!!!!!!!!")
	log.Print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	log.Printf("test: %s", ctx.Conn)

	return nil
}

func (t *Test) TestJoin(ctx *server.Context) error {
	log.Printf("test: Yeah! I joined the channel id: %s", ctx.Params("id"))

	ctx.Send("ack", map[string]interface{}{
		"yep": "connected",
	})

	return nil
}

func (t *Test) Leave(ctx *server.Context) error {
	log.Printf("test: Conn leaving %s", ctx.Conn)

	ctx.Broadcast("person-left", map[string]interface{}{
		"id":            ctx.Conn.Id,
		"what-happened": "they left",
	})

	return nil
}

func (t *Test) OtherTestHandler(ctx *server.Context) error {
	var testData TestData

	ctx.Data(&testData)

	log.Printf("OTHER: %s: %v", ctx.Channel.Name, ctx.Params("id"))
	log.Printf("My Test %#v", testData)

	return nil
}
