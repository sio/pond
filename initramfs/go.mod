module github.com/sio/pond/initramfs

go 1.20

require (
	github.com/klauspost/compress v1.17.6
	github.com/sio/pond/lib/sandbox v0.0.0
	golang.org/x/sys v0.17.0
)

require (
	github.com/vishvananda/netlink v1.1.0 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
)

replace github.com/sio/pond/lib/sandbox => ../lib/sandbox
