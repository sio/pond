package main

import (
	"secrets/server"

	"context"
	"log"
)

func main() {
	s, e := server.New("tests/keys/storage.pub")
	if e != nil {
		log.Fatal(e)
	}
	s.Run(context.Background(), ":2222")
}
