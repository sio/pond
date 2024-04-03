package main

import (
	"context"
	"log"
	"os"

	"github.com/sio/pond/nbd/server"
)

func main() {
	s := server.New(context.Background(), func(name string) (server.Backend, error) {
		log.Printf("exportFunc: client requested export name: %q\n", name)
		return os.Open("Makefile")
	})
	go s.ListenShutdown()
	e := s.Listen("tcp", "127.0.0.189:10809")
	if e != nil {
		log.Fatalf("Exit: %v", e)
	}
}
