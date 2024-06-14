package main

import (
	"fmt"
	"math/rand/v2"
	"time"
	"os"
)

func main() {
	number := rand.N(1000)
	fmt.Printf("NEEDLE=%v\n", number)
	fmt.Printf("PID=%v\n", os.Getpid())
	go sleep()
	select {}
	fmt.Println(number) // avoid being collected as garbage
}

func sleep() {
	for {
		time.Sleep(time.Minute)
	}
}
