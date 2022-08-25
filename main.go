package main

import (
	"log"
	"net/http"
	"os"

	"github.com/colevoss/awesome-go-realtime/server"
	"github.com/colevoss/awesome-go-realtime/test"
)

var addr = "localhost:8080"

func run() error {
	rtServer := server.NewRealtimeServer()

	// cf := server.NewChannelFactory("test.{id}.channel")

	otherCf := server.NewChannelFactory("test.channel")

	otherCf.Handle("test", func(c *server.Context) error {
		c.Send("test", map[string]interface{}{
			"yeah": "yeah",
		})

		return nil
	})

	test := &test.Test{}

	rtServer.RegisterChannel("test.{id}.channel", test)

	rtServer.Hub.RegisterChannelFactory(otherCf)

	go rtServer.Start()

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
