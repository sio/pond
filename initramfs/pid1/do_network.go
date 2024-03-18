package pid1

import (
	"github.com/sio/pond/initramfs/net"
)

func networkUp() error {
	return net.Up("eth0") // TODO
}
