package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAppleProfileUsesOnlyTheSelectedProvidersVerifiedServers(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "all-providers-results.json")
	rows := []map[string]string{
		profileTestRow("vpn unlimited", "slow.example.com", "192.0.2.1", "88", statusOK),
		profileTestRow("vpn unlimited", "partial.example.com", "192.0.2.2", "1", statusNoReply),
		profileTestRow("nordvpn", "se1.nordvpn.com", "192.0.2.3", "2", statusOK),
		profileTestRow("vpn unlimited", "fast.example.com", "192.0.2.4", "0", statusOK),
		profileTestRow("vpn unlimited", "duplicate.example.com", "192.0.2.4", "5", statusOK),
	}
	writeTestReport(t, report, "NordVPN and gluetun-servers", rows)
	account := vpnAccount{Alias: "office", Provider: "vpn unlimited", Username: "test-user", Password: "fixture<&>password"}
	output := filepath.Join(dir, "office.mobileconfig")
	opts := profileOptions{input: report, output: output, limit: 4}
	path, count, err := makeAppleProfile(account, opts)
	if err != nil || path != output || count != 2 {
		t.Fatalf("generated %d VPN entries in %q: %v", count, path, err)
	}
	pl, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	validateXML(t, pl)
	fast := bytes.Index(pl, []byte("fast.example.com"))
	slow := bytes.Index(pl, []byte("slow.example.com"))
	if fast < 0 || slow <= fast || bytes.Contains(pl, []byte("partial.example.com")) || bytes.Contains(pl, []byte("se1.nordvpn.com")) {
		t.Fatal("profile includes the wrong VPN entries or does not sort by IKEv2 latency")
	}
	if bytes.Count(pl, []byte("<string>ironnodes.com</string>")) != 2 || bytes.Contains(pl, []byte(account.Password)) {
		t.Fatal("profile did not preserve the remote ID and safely encode the password")
	}
	if info, err := os.Stat(output); err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Fatalf("profile must be stored privately: %v", err)
	}
	if _, _, err := makeAppleProfile(account, opts); err != nil {
		t.Fatal(err)
	}
	reissued, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(pl, reissued) {
		t.Fatal("rebuilding the same profile must preserve its identities")
	}
}

func TestProfilesAcceptLegacyProviderReports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nord-results.json")
	rows := []map[string]string{
		profileTestRow("", "se1.nordvpn.com", "192.0.2.1", "4", statusOK),
		profileTestRow("", "other.example.com", "192.0.2.2", "1", statusOK),
	}
	writeTestReport(t, path, "cache example", rows)
	servers, err := profileServers(path, "nordvpn", "", 8)
	if err != nil || len(servers) != 1 || servers[0].remoteID != "se1.nordvpn.com" {
		t.Fatalf("legacy report did not identify its NordVPN server: %v", err)
	}
	if _, err := profileServers(path, "protonvpn", "", 8); err == nil {
		t.Fatal("expected an error when the report has no verified server from the requested provider")
	}
}

func TestAccountAliasesAndConfigDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VPNBAN_CONFIG_DIR", "")
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("AppData", root)
	original, err := accountDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"nord-home", "nord-work"} {
		account := vpnAccount{Alias: alias, Provider: "nordvpn", Username: alias, Password: "fixture-" + alias}
		if err := saveAccount(original, account); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := accountPath(original, "../outside"); err == nil {
		t.Fatal("invalid account aliases must be rejected")
	}
	moved := filepath.Join(root, "new-config")
	if _, err := setAccountDir(moved); err != nil {
		t.Fatal(err)
	}
	active, err := accountDir()
	if err != nil || active != moved {
		t.Fatalf("active account path = %q: %v", active, err)
	}
	saved, err := listAccounts(active)
	if err != nil || len(saved) != 2 {
		t.Fatalf("expected two accounts in the new directory: %v", err)
	}
	for _, summary := range saved {
		account, err := loadAccount(active, summary.Alias)
		if err != nil || account.Password != "fixture-"+summary.Alias {
			t.Fatalf("account %s lost its credentials: %v", summary.Alias, err)
		}
	}
}

func profileTestRow(provider, hostname, ip, rtt, phase2 string) map[string]string {
	return map[string]string{"provider": provider, "hostname": hostname, "ip": ip, "ike_ms": rtt,
		"ike_phase1": statusOK, "ike_phase2": phase2}
}

func writeTestReport(t *testing.T, path, source string, rows []map[string]string) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"source": source, "servers": rows})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func validateXML(t *testing.T, content []byte) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(content))
	for {
		if _, err := dec.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
	}
	if !strings.Contains(string(content), "<plist version=\"1.0\">") {
		t.Fatal("generated XML is missing the plist root")
	}
}
