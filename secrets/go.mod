module github.com/sio/pond/secrets

go 1.20

require (
	github.com/sio/pond/lib/sandbox v0.0.0
	github.com/alecthomas/kong v0.8.0
	golang.org/x/crypto v0.12.0
)

require golang.org/x/sys v0.11.0 // indirect

replace github.com/sio/pond/lib/sandbox => ../lib/sandbox
