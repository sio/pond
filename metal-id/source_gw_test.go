package metal_id

import (
	"testing"
)

func TestGatewayDiscovery(t *testing.T) {
	gw, err := gateway()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Gateway address(es): %v", gw)
	for _, ip := range gw {
		mac, err := neighbor(ip)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("MAC address for %v => %v", ip, mac)
	}
}
