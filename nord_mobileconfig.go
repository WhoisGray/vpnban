package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

type profileServer struct {
	row    map[string]string
	rtt    float64
	subnet string
}

// readNordProfileJSON accepts both vpnban reports and arrays of server objects.
func readNordProfileJSON(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Nord JSON: %w", err)
	}
	var raw json.RawMessage = data
	if len(bytes.TrimSpace(data)) > 0 && bytes.TrimSpace(data)[0] == '{' {
		var report struct {
			Servers json.RawMessage `json:"servers"`
		}
		if err := json.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("invalid Nord JSON: %w", err)
		}
		raw = report.Servers
	}
	var objects []map[string]json.RawMessage
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("Nord JSON must contain a servers array")
	}
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil, fmt.Errorf("invalid Nord servers array: %w", err)
	}
	rows := make([]map[string]string, 0, len(objects))
	for _, obj := range objects {
		row := make(map[string]string, len(obj))
		for k, v := range obj {
			var s string
			if json.Unmarshal(v, &s) == nil {
				row[k] = s
				continue
			}
			var list []string
			if json.Unmarshal(v, &list) == nil {
				row[k] = strings.Join(list, ",")
				continue
			}
			row[k] = string(v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func selectNordProfileServers(rows []map[string]string, max int) []profileServer {
	var candidates []profileServer
	for _, row := range rows {
		if !strings.EqualFold(row["status"], "online") || row["ip"] == "" || row["hostname"] == "" || row["city"] == "" {
			continue
		}
		if row["ike_phase1"] != "" || row["ike_phase2"] != "" {
			if ikePhaseRank(row["ike_phase1"], row["ike_phase2"]) != 0 {
				continue
			}
		} else if row["ike"] != "real" && row["proven"] != "ike" && !strings.Contains(strings.ToLower(row["technologies"]), "ikev2") {
			continue
		}
		var rtt float64
		for _, key := range []string{"ike_ms", "rtt_ms"} {
			n, err := strconv.ParseFloat(row[key], 64)
			if err == nil && n > 0 && n < 5000 {
				rtt = n
				break
			}
		}
		ip := net.ParseIP(row["ip"])
		if rtt == 0 || ip == nil {
			continue
		}
		subnet := ip.String()
		if v4 := ip.To4(); v4 != nil {
			subnet = fmt.Sprintf("%d.%d.%d", v4[0], v4[1], v4[2])
		}
		candidates = append(candidates, profileServer{row, rtt, subnet})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].rtt < candidates[j].rtt })
	if max == 0 || len(candidates) <= max {
		return candidates
	}
	// Prefer distinct IPv4 /24s within each city, then fill from the remaining servers.
	counts := map[string]int{}
	used := map[string]bool{}
	picked := map[int]bool{}
	var selected []profileServer
	for pass := 0; pass < 2 && len(selected) < max; pass++ {
		for i, s := range candidates {
			city := strings.ToLower(s.row["country_code"] + "|" + s.row["country"] + "|" + s.row["city"])
			subnet := city + "|" + s.subnet
			if picked[i] || counts[city] >= 2 || (pass == 0 && used[subnet]) {
				continue
			}
			picked[i] = true
			counts[city]++
			used[subnet] = true
			selected = append(selected, s)
			if len(selected) == max {
				break
			}
		}
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].rtt < selected[j].rtt })
	return selected
}

func profileUUID() string { return strings.ToUpper(uuid.NewString()) }
func profileFlag(code string) string {
	code = strings.ToUpper(code)
	if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return "🌐"
	}
	return string([]rune{0x1F1E6 + rune(code[0]-'A'), 0x1F1E6 + rune(code[1]-'A')})
}
func profileName(prefix string, s map[string]string) string {
	city := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			return r
		}
		return -1
	}, s["city"])
	host := strings.TrimSuffix(s["hostname"], ".nordvpn.com")
	return prefix + "-" + profileFlag(s["country_code"]) + city + "-" + host
}

type plistDict []plistEntry
type plistEntry struct {
	key   string
	value any
}

func plistXML(b *bytes.Buffer, value any) {
	switch v := value.(type) {
	case plistDict:
		b.WriteString("<dict>\n")
		for _, e := range v {
			b.WriteString("<key>")
			xml.EscapeText(b, []byte(e.key))
			b.WriteString("</key>\n")
			plistXML(b, e.value)
		}
		b.WriteString("</dict>\n")
	case []plistDict:
		b.WriteString("<array>\n")
		for _, e := range v {
			plistXML(b, e)
		}
		b.WriteString("</array>\n")
	case []byte:
		fmt.Fprintf(b, "<data>%s</data>\n", base64.StdEncoding.EncodeToString(v))
	case int:
		fmt.Fprintf(b, "<integer>%d</integer>\n", v)
	case string:
		b.WriteString("<string>")
		xml.EscapeText(b, []byte(v))
		b.WriteString("</string>\n")
	}
}
func writeNordMobileconfig(path string, rows []map[string]string, username, password, prefix string, max int) error {
	if !validPlistText(username) || !validPlistText(password) {
		return fmt.Errorf("service credentials contain characters that XML cannot represent")
	}
	servers := selectNordProfileServers(rows, max)
	if len(servers) == 0 {
		return fmt.Errorf("no suitable online IKEv2 servers with latency found for mobileconfig")
	}
	var payloads []plistDict
	for _, s := range servers {
		id := profileUUID()
		sa := plistDict{{"EncryptionAlgorithm", "AES-256"}, {"IntegrityAlgorithm", "SHA2-256"}, {"DiffieHellmanGroup", 14}, {"LifeTimeInMinutes", 1440}}
		ike := plistDict{{"RemoteAddress", s.row["ip"]}, {"RemoteIdentifier", s.row["hostname"]}, {"LocalIdentifier", ""}, {"AuthenticationMethod", "None"}, {"ExtendedAuthEnabled", 1}, {"AuthName", username}, {"AuthPassword", password}, {"DeadPeerDetectionRate", "Medium"}, {"DisableMOBIKE", 0}, {"DisableRedirect", 0}, {"EnableCertificateRevocationCheck", 0}, {"EnablePFS", 1}, {"UseConfigurationAttributeInternalIPSubnet", 0}, {"IKESecurityAssociationParameters", sa}, {"ChildSecurityAssociationParameters", sa}}
		payloads = append(payloads, plistDict{{"PayloadIdentifier", "com.nordvpn.ikev2." + id}, {"PayloadUUID", id}, {"PayloadType", "com.apple.vpn.managed"}, {"PayloadVersion", 1}, {"UserDefinedName", profileName(prefix, s.row)}, {"VPNType", "IKEv2"}, {"IKEv2", ike}, {"IPv4", plistDict{{"OverridePrimary", 1}}}, {"Proxies", plistDict{{"HTTPEnable", 0}, {"HTTPSEnable", 0}}}})
	}
	id := profileUUID()
	profile := plistDict{{"PayloadDisplayName", fmt.Sprintf("NordVPN IKEv2 (%d servers)", len(servers))}, {"PayloadIdentifier", "com.nordvpn.ikev2.profile." + id}, {"PayloadUUID", id}, {"PayloadType", "Configuration"}, {"PayloadVersion", 1}, {"PayloadContent", payloads}}
	if err := writeApplePlist(path, profile); err != nil {
		return fmt.Errorf("write mobileconfig: %w", err)
	}
	fmt.Fprintf(os.Stderr, "mobileconfig: %d servers written to %s\n", len(servers), path)
	return nil
}

func writeApplePlist(path string, profile plistDict) error {
	var out bytes.Buffer
	out.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n")
	plistXML(&out, profile)
	out.WriteString("</plist>\n")
	return writePrivateFile(path, out.Bytes())
}
