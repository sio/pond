package main

import (
	"log"

	"secrets/journal"
)

func main() {
	a, e := journal.Open(`C:\Temp\journal.log`)
	if e != nil {
		log.Fatal(e)
	}
	defer a.Close()
	a.Message(journal.Add, "hello", "world")
}
