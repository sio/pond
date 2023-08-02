package main

import (
	"log"

	"secrets/crypto"
	"secrets/journal"
)

func main() {
	k, e := crypto.LocalKey(`tests\keys\storage`)
	if e != nil {
		log.Fatal(e)
	}
	a, e := journal.Open(`C:\Temp\journal.log`, k)
	if e != nil {
		log.Fatal(e)
	}
	defer a.Close()
	a.Message(journal.Add, "hello", "world")
}
