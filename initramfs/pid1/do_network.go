package pid1

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
	"github.com/vishvananda/netlink"
)

// Try to configure all interfaces at once, stop at first success
func networkUp() error {
	ctx, cancel := context.WithCancel(context.Background())
	interfaces, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return err
	}
	errs := make(map[string]error)
	tick := make(chan struct{})
	for _, iface := range interfaces {
		go func(iface string) {
			defer func() { tick <- struct{}{} }()
			err := configure(ctx, iface)
			if err == nil {
				cancel()
				return
			}
			errs[iface] = err
		}(iface.Name())
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick:
			if len(interfaces) == 0 {
				return fmt.Errorf("no network interfaces in /sys/class/net")
			}
			if len(errs) == len(interfaces) {
				message := new(strings.Builder)
				for _, iface := range interfaces {
					_, _ = fmt.Fprintf(message, "%s: %s\n", iface.Name(), errs[iface.Name()])
				}
				return fmt.Errorf("all interfaces failed:\n%s", strings.TrimSpace(message.String()))
			}
		}
	}
}

// Configure a single network interface
func configure(ctx context.Context, iface string) error {

	// $ ip link set $IFACE up
	link, err := ifup(iface)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// $ dhclient -v $IFACE
	config, err := dhcpc(ctx, iface)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// $ ip addr add $ADDRESS/$NETMASK dev $IFACE
	err = netlink.AddrAdd(
		link,
		&netlink.Addr{
			IPNet: &net.IPNet{config.addr, config.subnet},
		},
	)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// $ ip route add default via $ROUTER dev $IFACE
	route := &netlink.Route{
		Dst:       &net.IPNet{},
		Gw:        config.router,
		LinkIndex: link.Attrs().Index,
	}
	return netlink.RouteAdd(route)
}

// Bring interface up
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
		return link, fmt.Errorf("loopback interface; no outside connectivity")
	}
	return link, netlink.LinkSetUp(link)
}

// Network settings we care about
type network struct {
	addr   net.IP
	router net.IP
	subnet net.IPMask
	dns    []net.IP
}

// Obtain DHCP lease
func dhcpc(ctx context.Context, iface string) (settings network, err error) {
	client, err := nclient4.New(iface)
	if err != nil {
		return settings, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	offer, err := client.DiscoverOffer(ctx)
	if err != nil {
		return settings, fmt.Errorf("DHCP on %s: %w", iface, err)
	}
	lease, err := client.RequestFromOffer(ctx, offer)
	if err != nil {
		return settings, fmt.Errorf("DHCP on %s: %w", iface, err)
	}
	ack := lease.ACK

	router := dhcpv4.GetIP(dhcpv4.OptionRouter, ack.Options)
	if empty(router) {
		router = ack.GatewayIPAddr
	}
	if empty(router) {
		router = ack.ServerIPAddr
	}
	if empty(router) {
		return settings, fmt.Errorf("DHCP on %s: could not detect router address", iface)
	}
	settings.router = router

	settings.addr = ack.YourIPAddr
	if empty(settings.addr) {
		return settings, fmt.Errorf("DHCP on %s: did not receive an IP address", iface)
	}

	subnetBytes := ack.Options.Get(dhcpv4.OptionSubnetMask)
	settings.subnet = net.IPMask(subnetBytes)
	ones, bits := settings.subnet.Size()
	if subnetBytes == nil || (ones+bits) == 0 {
		settings.subnet = settings.addr.DefaultMask()
	}

	settings.dns = dhcpv4.GetIPs(dhcpv4.OptionDomainNameServer, ack.Options)
	if len(settings.dns) == 0 {
		settings.dns = []net.IP{
			settings.router,
			net.ParseIP("8.8.8.8"),
			net.ParseIP("8.8.4.4"),
		}
		if !empty(ack.ServerIPAddr) {
			settings.dns = append(settings.dns, ack.ServerIPAddr)
		}
	}
	return settings, nil
}

func empty(addr net.IP) bool {
	return addr.IsUnspecified() || addr == nil
}
