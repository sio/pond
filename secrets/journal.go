package main

import (
	"log"

	"secrets/crypto"
	"secrets/journal"
)

func main() {
	k, e := crypto.LocalKey(`tests\keys\storage`)
	if e != nil {
		log.Println(e)
		return
	}
	a, e := journal.Open(`C:\Temp\journal.log`, k)
	if e != nil {
		log.Println(e)
		return
	}
	defer a.Close()
	e = a.CatchUp()
	if e != nil {
		log.Println(e)
		return
	}
	e = a.Message(journal.Add, "hello", "world")
	if e != nil {
		log.Println(e)
		return
	}
}
