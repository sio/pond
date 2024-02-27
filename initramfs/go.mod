module github.com/sio/pond/initramfs

go 1.20

require (
	github.com/klauspost/compress v1.17.6
	github.com/sio/pond/lib/sandbox v0.0.0
	golang.org/x/sys v0.17.0
)

replace github.com/sio/pond/lib/sandbox => ../lib/sandbox
