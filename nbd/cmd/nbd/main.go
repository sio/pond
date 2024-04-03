package main

import (
	"context"
	"log"
	"os"

	"github.com/sio/pond/nbd/server"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	s := server.New(context.Background(), func(name string) (server.Backend, error) {
		log.Printf("exportFunc: client requested export name: %q\n", name)
		return os.Open(exe)
	})
	go s.ListenShutdown()
	e := s.Listen("tcp", "127.0.0.189:10809")
	if e != nil {
		log.Fatalf("Exit: %v", e)
	}
}
