module github.com/sio/pond/initramfs

go 1.20

require (
	github.com/insomniacslk/dhcp v0.0.0-20240227161007-c728f5dd21c8
	github.com/klauspost/compress v1.17.6
	github.com/sio/pond/lib/sandbox v0.0.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/sys v0.17.0
)

require (
	github.com/josharian/native v1.1.0 // indirect
	github.com/mdlayher/packet v1.1.2 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.14 // indirect
	github.com/u-root/uio v0.0.0-20230220225925-ffce2a382923 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
)

replace github.com/sio/pond/lib/sandbox => ../lib/sandbox
