package main

import (
	"secrets/server"

	"context"
	"log"
)

func main() {
	s, e := server.New("tests/keys/storage.pub", "/tmp/hello.sqlite")
	if e != nil {
		log.Fatal(e)
	}
	s.Run(context.Background(), "127.0.0.1:2222")
}
