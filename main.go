package main

import (
	"chatBackend/auth"
	"chatBackend/config"
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/nats-io/nats.go"
)

var ctx = context.Background()

func main() {
	flag.Parse()

	config.CreateRedisClient()

	uriNats := os.Getenv("NATS_URI")
	nc, err := nats.Connect(uriNats)

	if err != nil {
		panic(err)
	}

	go metrics()

	server := NewWebsocketServer(nc, config.Redis)
	go server.Run()

	http.HandleFunc("/ws", auth.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(server, w, r)
	}))

	api := &API{redis: config.Redis}
	http.HandleFunc("/api/register", api.HandleRegister)
	http.HandleFunc("/api/login", api.HandleLogin)

	addr := os.Getenv("ADDR")
	log.Printf("Running on port %s", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}
