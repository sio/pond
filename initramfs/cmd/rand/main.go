// Sample application for testing random generator with dieharder
//
// $ rand | dieharder -g 200 -a
package main

import (
	"github.com/sio/pond/initramfs/rand"
	"os"
)

func main() {
	buf := make([]byte, os.Getpagesize())
	for {
		rand.Seed(buf)
		_, err := os.Stdout.Write(buf)
		if err != nil {
			panic(err)
		}
	}
}
