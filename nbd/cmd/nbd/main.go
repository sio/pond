package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sio/pond/nbd/server"
)

func main() {
	s := server.New(context.Background(), func(name string) (io.ReaderAt, error) {
		fmt.Printf("Requested export: %q\n", name)
		return os.Open("Makefile")
	})
	e := s.Listen("tcp", "127.0.0.189:10809")
	fmt.Printf("Exit: %v", e)
}
