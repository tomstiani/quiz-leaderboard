package server

import (
	"net"
	"testing"
)

func TestPublicIPRejectsInternalNetworks(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "100.64.0.1", "172.16.0.1", "192.0.2.1", "192.168.1.1", "198.18.0.1", "198.51.100.1", "203.0.113.1", "169.254.1.1", "::1", "2001:db8::1", "fc00::1", "fe80::1"} {
		if publicIP(net.ParseIP(address)) {
			t.Fatalf("accepted internal address %s", address)
		}
	}
	for _, address := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !publicIP(net.ParseIP(address)) {
			t.Fatalf("rejected public address %s", address)
		}
	}
}
