package main

import (
	"log"

	"secrets/audit"
)

func main() {
	a, e := audit.Open(`C:\Temp\audit.log`)
	if e != nil {
		log.Fatal(e)
	}
	defer a.Close()
	a.Message(audit.Add, "hello", "world")
}
