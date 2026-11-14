package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNordMobileconfigSelection(t *testing.T) {
	rows := []map[string]string{}
	for _, x := range []struct{ city, ip, rtt string }{{"A", "1.2.3.1", "10"}, {"A", "1.2.3.2", "11"}, {"A", "1.2.4.1", "12"}, {"B", "1.2.5.1", "13"}} {
		rows = append(rows, map[string]string{"city": x.city, "ip": x.ip, "rtt_ms": x.rtt, "status": "online", "hostname": "test.nordvpn.com", "ike": "real", "country_code": "US"})
	}
	if got := selectNordProfileServers(rows, 4); len(got) != 4 {
		t.Fatalf("without overflow got %d", len(got))
	}
	got := selectNordProfileServers(rows, 3)
	if len(got) != 3 || got[0].row["ip"] != "1.2.3.1" || got[1].row["ip"] != "1.2.4.1" || got[2].row["city"] != "B" {
		t.Fatalf("selection: %+v", got)
	}
	if got := selectNordProfileServers(rows, 0); len(got) != 4 {
		t.Fatal("unlimited should retain all")
	}
}

func TestNordMobileconfigFromJSON(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "results.json")
	output := filepath.Join(dir, "profile.mobileconfig")
	data := `{"servers":[{"city":"Paris","country_code":"FR","ip":"1.2.3.4","hostname":"fr1.nordvpn.com","status":"online","technologies":["ikev2"],"ike_ms":42}]}`
	if err := os.WriteFile(input, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VPNBAN_NORD_USERNAME", "user&name")
	t.Setenv("VPNBAN_NORD_PASSWORD", "pass<word")
	opts := nordOpts{inputPath: input, mobileconfigPath: output, maxServers: 26, prefixConfig: "$I"}
	if err := runNord(opts); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := xml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	for _, want := range []string{"user&amp;name", "pass&lt;word", "$I-🇫🇷Paris-fr1", "com.apple.vpn.managed"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %q", want)
		}
	}
	opts.username, opts.password = "flag-user", "flag-pass"
	opts.saw = map[string]bool{"nord-username": true, "nord-password": true}
	if err := runNord(opts); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "flag-user") || !strings.Contains(string(b), "flag-pass") || strings.Contains(string(b), "user&amp;name") {
		t.Fatal("credential flags did not override environment")
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v %v", info, err)
	}
	opts.mobileconfigPath = input
	if err := runNord(opts); err == nil {
		t.Fatal("input must not be overwritten")
	}
}

func TestNordMobileconfigMissingCredentialsAndServers(t *testing.T) {
	t.Setenv("VPNBAN_NORD_USERNAME", "")
	t.Setenv("VPNBAN_NORD_PASSWORD", "")
	if err := runNord(nordOpts{inputPath: "anything"}); err == nil {
		t.Fatal("expected credentials error")
	}
	if err := writeNordMobileconfig(filepath.Join(t.TempDir(), "out"), nil, "a", "b", "", 26); err == nil {
		t.Fatal("expected no servers error")
	}
}
