package main

import (
	"testing"
	"time"
)

func TestNordGluetunFallback(t *testing.T) {
	c := gluetunCatalog{Servers: []gluetunServer{
		{Hostname: "us1.nordvpn.com", Country: "United States", VPN: "openvpn", UDP: true, TCP: true, IPs: []string{"8.8.8.8", "10.0.0.1"}},
		{Hostname: "us1.nordvpn.com", Country: "United States", VPN: "wireguard", WGPubKey: "test-key", IPs: []string{"8.8.8.8"}},
	}}
	list := nordFromGluetun(c)
	if len(list) != 1 {
		t.Fatalf("got %d endpoints, want one public endpoint", len(list))
	}
	s := list[0]
	if !s.Fallback || s.Status != "unknown" || s.Load != -1 || s.WGPub != "test-key" {
		t.Fatalf("unexpected server: %+v", s)
	}
	for _, tech := range []string{"wireguard_udp", "openvpn_udp", "openvpn_tcp"} {
		if !hasTech(s.Tech, tech) {
			t.Errorf("missing %s", tech)
		}
	}
	if !countryOK(s, []string{"us"}) {
		t.Fatal("US country filter did not match")
	}
	scan, err := parseScanProtos("ikev2")
	if err != nil || !scan.match(s) {
		t.Fatal("fallback should allow IKEv2 discovery")
	}
	path := t.TempDir() + "/cache.json"
	if err := writeNordCache(path, list); err != nil {
		t.Fatal(err)
	}
	cached, _, ok := readNordCache(path, time.Hour, time.Now())
	if !ok || !cached[0].Fallback {
		t.Fatal("fallback provenance was not cached")
	}
}
