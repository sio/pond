package pid1

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func ifup(iface string) (link netlink.Link, err error) {
	link, err = netlink.LinkByName(iface)
	if err != nil {
		return nil, err
	}
	attrs := link.Attrs()
	if attrs.Flags&net.FlagUp != 0 {
		return link, nil // already up
	}
	if attrs.Flags&net.FlagLoopback != 0 {
		return link, fmt.Errorf("loopback interfaces are not supported: %s", iface)
	}
	return link, netlink.LinkSetUp(link)
}

func ip(link netlink.Link, addr string) error {
	address, err := netlink.ParseAddr(addr)
	if err != nil {
		return err
	}
	return netlink.AddrAdd(link, address)
}

func route(link netlink.Link, gw net.IP) error {
	_, cidr, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		return err
	}
	route := &netlink.Route{
		Dst:       cidr,
		Gw:        gw,
		LinkIndex: link.Attrs().Index,
	}
	return netlink.RouteAdd(route)
}

func networkUp() error {
	link, err := ifup("eth0") // TODO
	if err != nil {
		return err
	}
	err = ip(link, "10.0.2.15/24") // TODO
	if err != nil {
		return err
	}
	return route(link, net.ParseIP("10.0.2.2")) // TODO
}
