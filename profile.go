package main

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const defaultProfileLimit = 8

type profileOptions struct {
	input, output, ca, remoteID string
	limit                       int
}

type appleEndpoint struct {
	hostname, ip, remoteID string
	rtt                    int
}

func defaultReport(provider string) string {
	name := strings.ReplaceAll(provider, " ", "-") + "-results.json"
	if provider == "nordvpn" {
		name = "nord-results.json"
	}
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat("all-providers-results.json"); err == nil {
			return "all-providers-results.json"
		}
	}
	return name
}

func legacyReportProvider(source string) string {
	if rest, ok := strings.CutPrefix(source, "gluetun-servers/"); ok {
		name, _, _ := strings.Cut(rest, " (")
		return name
	}
	if strings.HasPrefix(source, "gluetun-servers fallback (") {
		return "nordvpn"
	}
	if prefix, _, _ := strings.Cut(source, " "); prefix == "download" || prefix == "cache" || prefix == "stale" {
		return "nordvpn"
	}
	return ""
}

func profileServers(input, provider, remoteID string, limit int) ([]appleEndpoint, error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var report struct {
		Source  string              `json:"source"`
		Servers []map[string]string `json:"servers"`
	}
	if err := json.NewDecoder(f).Decode(&report); err != nil {
		return nil, fmt.Errorf("read provider report: %w", err)
	}
	legacy := legacyReportProvider(report.Source)
	servers := make([]appleEndpoint, 0, min(limit, len(report.Servers)))
	for _, row := range report.Servers {
		if or(row["provider"], legacy) != provider || ikePhaseRank(row["ike_phase1"], row["ike_phase2"]) != 0 {
			continue
		}
		ip, err := netip.ParseAddr(row["ip"])
		host, ok := normalizeHost(row["hostname"])
		if err != nil || ip.Zone() != "" || !ok {
			continue
		}
		if provider == "nordvpn" && !strings.HasSuffix(host, ".nordvpn.com") {
			continue
		}
		id := or(row["ike_remote_id"], host)
		if provider == "vpn unlimited" && row["ike_remote_id"] == "" {
			id = "ironnodes.com"
		}
		id = or(remoteID, id)
		if !validPlistText(id) {
			return nil, errors.New("remote ID contains characters that XML cannot represent")
		}
		rtt, err := strconv.Atoi(or(row["ike_ms"], row["rtt_ms"]))
		if err != nil || rtt < 0 {
			rtt = math.MaxInt
		}
		servers = append(servers, appleEndpoint{hostname: host, ip: ip.String(), remoteID: id, rtt: rtt})
	}
	sort.SliceStable(servers, func(i, j int) bool { return servers[i].rtt < servers[j].rtt })
	selected := servers[:0]
	seen := make(map[string]bool)
	for _, server := range servers {
		key := server.ip + "|" + server.remoteID
		if seen[key] {
			continue
		}
		seen[key] = true
		selected = append(selected, server)
		if len(selected) == limit {
			break
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no %s servers passed both IKEv2 phases in %s", provider, input)
	}
	return selected, nil
}

func makeAppleProfile(account vpnAccount, opts profileOptions) (string, int, error) {
	if opts.limit < 1 {
		return "", 0, errors.New("profile -limit must be at least 1")
	}
	if !validPlistText(account.Username) || !validPlistText(account.Password) {
		return "", 0, errors.New("service credentials contain characters that XML cannot represent")
	}
	input := or(opts.input, defaultReport(account.Provider))
	output := or(opts.output, account.Alias+".mobileconfig")
	if !strings.EqualFold(filepath.Ext(output), ".mobileconfig") {
		return "", 0, errors.New("profile output must end in .mobileconfig")
	}
	servers, err := profileServers(input, account.Provider, opts.remoteID, opts.limit)
	if err != nil {
		return "", 0, err
	}
	identifier := "com.vpnban.profile." + account.Alias
	var payloads []plistDict
	if opts.ca != "" {
		cert, err := loadProfileCA(opts.ca)
		if err != nil {
			return "", 0, err
		}
		ca := profilePayload("com.apple.security.root", identifier+".ca", "VPN root certificate")
		ca = append(ca, plistEntry{"PayloadCertificateFileName", "vpn-root.cer"}, plistEntry{"PayloadContent", cert})
		payloads = append(payloads, ca)
	}
	for _, server := range servers {
		name := account.Alias + " · " + server.hostname
		id := identifier + ".vpn." + strings.ToLower(appleProfileUUID(server.ip+"|"+server.remoteID))
		vpn := profilePayload("com.apple.vpn.managed", id, name)
		ike := plistDict{
			{"RemoteAddress", server.ip}, {"RemoteIdentifier", server.remoteID}, {"LocalIdentifier", ""},
			{"AuthenticationMethod", "None"}, {"ExtendedAuthEnabled", 1},
			{"AuthName", account.Username}, {"AuthPassword", account.Password},
		}
		vpn = append(vpn,
			plistEntry{"UserDefinedName", name},
			plistEntry{"VPNType", "IKEv2"},
			plistEntry{"IPv4", plistDict{{"OverridePrimary", 1}}},
			plistEntry{"IKEv2", ike},
		)
		payloads = append(payloads, vpn)
	}
	profile := profilePayload("Configuration", identifier, "vpnban · "+account.Alias)
	profile = append(profile, plistEntry{"PayloadOrganization", "vpnban"}, plistEntry{"PayloadContent", payloads})
	if err := writeApplePlist(output, profile); err != nil {
		return "", 0, err
	}
	return output, len(servers), nil
}

func loadProfileCA(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if block, _ := pem.Decode(data); block != nil {
		if block.Type != "CERTIFICATE" {
			return nil, errors.New("profile -ca requires a PEM or DER root certificate")
		}
		data = block.Bytes
	}
	cert, err := x509.ParseCertificate(data)
	if err != nil || !cert.IsCA || cert.CheckSignatureFrom(cert) != nil {
		return nil, errors.New("profile -ca requires a self-signed root CA certificate")
	}
	return cert.Raw, nil
}

func profilePayload(kind, id, title string) plistDict {
	return plistDict{
		{"PayloadType", kind}, {"PayloadIdentifier", id}, {"PayloadUUID", appleProfileUUID(id)},
		{"PayloadVersion", 1}, {"PayloadDisplayName", title},
	}
}

// UUIDv5 with the standard DNS namespace keeps the same identities on regeneration.
func appleProfileUUID(s string) string {
	namespace := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	sum := sha1.Sum(append(namespace, s...))
	sum[6] = sum[6]&0x0f | 0x50
	sum[8] = sum[8]&0x3f | 0x80
	return fmt.Sprintf("%X-%X-%X-%X-%X", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func validPlistText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0xFFFE || r == 0xFFFF {
			return false
		}
	}
	return true
}
