package main

import (
	"slices"
	"testing"
)

func TestProviderRankingPrefersBothPhasesThenIKELatency(t *testing.T) {
	hits := []nordHit{
		{Server: nordServer{Name: "phase-one"}, IKE: "p1", IKEMS: 1, Fresh: "yes", Firewall: "phase1"},
		{Server: nordServer{Name: "wireguard"}, WG: "real", WGMS: 2, Fresh: "yes", Firewall: "clean"},
		{Server: nordServer{Name: "slower", Load: 1}, IKE: "real", IKEMS: 45, WG: "real", WGMS: 3, Fresh: "yes", Firewall: "clean"},
		{Server: nordServer{Name: "faster", Load: 90}, IKE: "real", IKEMS: 18, Fresh: "yes", Firewall: "clean"},
	}
	sortNordHits(hits)
	got := make([]string, len(hits))
	for i, hit := range hits {
		got[i] = hit.Server.Name
	}
	want := []string{"faster", "slower", "wireguard", "phase-one"}
	if !slices.Equal(got, want) {
		t.Fatalf("server order = %v, want %v", got, want)
	}
}

func TestFileExportsRespectProbeOrder(t *testing.T) {
	fast, slow, instant := 15, 90, 1
	probes := []probe{
		{Domain: "partial.example", Phase1: statusOK, AvgMS: &instant},
		{Domain: "slow.example", Phase1: statusOK, Phase2: statusOK, AvgMS: &slow},
		{Domain: "fast.example", Phase1: statusOK, Phase2: statusOK, IKEMS: &fast},
	}
	entries := make([]entry, 0, len(probes)+1)
	for _, domain := range []string{"unresolved.example", "slow.example", "partial.example", "fast.example"} {
		entries = append(entries, entry{Domain: domain, Fields: map[string]string{"domain": domain}})
	}
	sortProbes(probes)
	rows := exportRows(entries, probes, nil)
	got := make([]string, len(rows))
	for i, row := range rows {
		got[i] = row["domain"]
	}
	want := []string{"fast.example", "slow.example", "partial.example", "unresolved.example"}
	if !slices.Equal(got, want) {
		t.Fatalf("export order = %v, want %v", got, want)
	}
}
