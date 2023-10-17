module github.com/sio/pond/secrets

go 1.20

require (
	github.com/alecthomas/kong v0.8.1
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/sio/pond/lib/block v0.0.0
	github.com/sio/pond/lib/bytepack v0.0.0
	github.com/sio/pond/lib/sandbox v0.0.0
	golang.org/x/crypto v0.14.0
)

require golang.org/x/sys v0.13.0 // indirect

replace (
	github.com/sio/pond/lib/block => ../lib/block
	github.com/sio/pond/lib/bytepack => ../lib/bytepack
	github.com/sio/pond/lib/sandbox => ../lib/sandbox
)
