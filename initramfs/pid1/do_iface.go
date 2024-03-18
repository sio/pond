package pid1

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Linux kernel interface flags
//
// https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/uapi/linux/if.h#n42
// https://utcc.utoronto.ca/~cks/space/blog/linux/SysfsNetworkInterfaceStatus
type iflag uint64

const (
	iff_UP iflag = 1 << iota
	iff_BROADCAST
	iff_DEBUG
	iff_LOOPBACK
	iff_POINTOPOINT
	iff_NOTRAILERS
	iff_RUNNING
	iff_NOARP
	iff_PROMISC
	iff_ALLMULTI
	iff_MASTER
	iff_SLAVE
	iff_MULTICAST
	iff_PORTSEL
	iff_AUTOMEDIA
	iff_DYNAMIC
	iff_LOWER_UP
	iff_DORMANT
	iff_ECHO
)

func ifup(iface string) error {
	flagPath := fmt.Sprintf("/sys/class/net/%s/flags", iface)
	raw, err := os.ReadFile(flagPath)
	if err != nil {
		return err
	}
	number, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 0, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", flagPath, err)
	}
	flags := iflag(number)
	if flags&iff_UP != 0 {
		return nil
	}
	if flags&iff_LOOPBACK != 0 {
		return fmt.Errorf("this tool is not intended for use with loopback interfaces: %s", iface)
	}
	flags |= iff_UP
	return os.WriteFile(flagPath, []byte(fmt.Sprintf("%x\n", flags)), 0)
}

func networkUp() error {
	return ifup("eth0") // TODO
}
