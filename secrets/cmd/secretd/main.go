package main

import (
	"secrets/db"

	"fmt"
)

func main() {
	secrets, err := db.Open("hello.sqlite")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer secrets.Close()
	fmt.Println("Secrets:", secrets)
}
