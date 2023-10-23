package metal_id

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type GatewayData struct {
	abstractDataSource
}

func (d *GatewayData) Next() []byte {
	if d.IsEmpty() {
		d.fill()
	}
	return d.abstractDataSource.Next()
}

func (d *GatewayData) fill() {
	gw, err := gateway()
	if err != nil {
		d.err = err
		return
	}
	for _, ip := range gw {
		mac, err := neighbor(ip)
		if err != nil {
			d.err = err
			return
		}
		d.Append([]byte(fmt.Sprintf("%v@%v", ip, mac)))
	}
}

func neighbor(ip net.IP) (net.HardwareAddr, error) {
	arp, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer func() { _ = arp.Close() }()
	scanner := bufio.NewScanner(arp)
	var header bool = true
	for scanner.Scan() {
		if header {
			header = false
			continue
		}
		field := strings.Fields(scanner.Text())
		if len(field) < 4 {
			return nil, fmt.Errorf("unexpected number of colums in arp table: %s", scanner.Text())
		}
		current := net.ParseIP(field[0])
		if !ip.Equal(current) {
			continue
		}
		return net.ParseMAC(field[3])
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return nil, fmt.Errorf("not found in arp table: %v", ip)
}

func gateway() ([]net.IP, error) {
	route, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer func() { _ = route.Close() }()
	const (
		// https://github.com/torvalds/linux/blob/05d3ef8bba77c1b5f98d941d8b2d4aeab8118ef1/include/uapi/linux/route.h#L51-L61
		flagUp      uint64 = 0b01
		flagGateway uint64 = 0b10
	)
	var gw []net.IP
	scanner := bufio.NewScanner(route)
	var header bool = true
	for scanner.Scan() {
		if header {
			header = false
			continue
		}
		field := strings.Fields(scanner.Text())
		if len(field) < 4 {
			return nil, fmt.Errorf("unexpected number of columns in route table: %s", scanner.Text())
		}
		flags, err := strconv.ParseUint(field[3], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse route flags: %w", err)
		}
		if flags&flagUp == 0 || flags&flagGateway == 0 {
			continue
		}
		ip, err := hex.DecodeString(field[2])
		if err != nil || len(ip) != 4 {
			return nil, fmt.Errorf("invalid value in IP field: %q", field[2])
		}
		gw = append(gw, net.IPv4(ip[3], ip[2], ip[1], ip[0])) // TODO: assumes big endian machine
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	if len(gw) == 0 {
		return nil, fmt.Errorf("failed to find gateway IP")
	}
	return gw, nil
}
