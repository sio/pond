package main

import (
	"fmt"

	"github.com/sio/pond/initramfs/pre"
)

func main() {
	fmt.Println("Hello from custom initramfs!")
	pre.Run()
}
