package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sio/pond/nbd/daemon"
	"github.com/sio/pond/nbd/logger"
)

const config = `
{
	"s3": {
		"endpoint": "http://127.0.0.55:55555",
		"bucket": "testdata",
		"access": "access",
		"secret": "secret123"
	},
	"cache": {"dir": "./cache"},
	"listen": [
		{"network": "tcp", "address": "127.0.0.189:10809"}
	]
}
`

func main() {
	logger.Setup()

	var nbd daemon.Daemon
	err := json.Unmarshal([]byte(config), &nbd)
	if err != nil {
		panic("default config: " + err.Error())
	}

	err = nbd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
