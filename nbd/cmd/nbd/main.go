package main

import (
	"context"
	"fmt"

	"github.com/sio/pond/nbd/server"
)

func main() {
	s := server.New(context.Background(), nil)
	e := s.Listen("tcp", "127.0.0.189:10809")
	fmt.Printf("Exit: %v", e)
}
