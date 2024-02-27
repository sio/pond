package main

import (
	"fmt"

	"github.com/sio/pond/initramfs/pid1"
)

func main() {
	fmt.Println("Hello from custom initramfs!")
	pid1.Run()
}
