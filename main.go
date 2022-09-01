package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/colevoss/awesome-go-realtime/server"
	"github.com/colevoss/awesome-go-realtime/test"
)

var addr = "localhost:8080"

func run() error {
	rtServer := server.NewRealtimeServer()

	otherCf := server.NewChannelFactory("test.channel")

	otherCf.Handle("test", func(ctx context.Context, c *server.Event) error {
		c.Send("test", map[string]interface{}{
			"yeah": "yeah",
		})

		return nil
	})

	test := &test.Test{}

	rtServer.RegisterChannel("test.{id}.channel", test)
	rtServer.RegisterChannelFactory(otherCf)

	http.Handle("/rt", rtServer)
	log.Println("Starting server:", addr)
	return http.ListenAndServe(addr, nil)
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
