package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"sort"
	"time"
)

var errNordLoad = errors.New("-load must be between 0 and 100")

type providerScan struct {
	hits    []nordHit
	source  string
	fetched time.Time
}

func (s providerScan) write(jsonPath, csvPath, xlsxPath string, includeProvider bool) error {
	sortNordHits(s.hits)
	rows, cols := flattenNord(s.hits, includeProvider)
	if err := writeNord(jsonPath, csvPath, xlsxPath, s.source, s.fetched, rows, cols); err != nil {
		return err
	}
	summarizeNord(s.hits)
	return nil
}

func sortNordHits(hits []nordHit) {
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		aVerified, bVerified := a.IKE == "real", b.IKE == "real"
		if aVerified != bVerified {
			return aVerified
		}
		if !aVerified {
			ar, br := freshRank(a.Fresh, a.Firewall), freshRank(b.Fresh, b.Firewall)
			if ar != br {
				return ar < br
			}
		}
		if ar, br := a.rtt(), b.rtt(); ar != br {
			return ar < br
		}
		if a.Server.Load != b.Server.Load {
			return a.Server.Load < b.Server.Load
		}
		return a.Server.Hostname < b.Server.Hostname
	})
}

func runAllProviders(nord nordOpts, gluetun gluetunOpts) error {
	protocols, err := parseScanProtos(nord.proto)
	if err != nil {
		return err
	}
	if nord.maxLoad < 0 || nord.maxLoad > 100 {
		return errNordLoad
	}
	if !nord.saw["json"] {
		nord.jsonPath = "all-providers-results.json"
	}
	if !nord.saw["csv"] {
		nord.csvPath = "all-providers-results.csv"
	}
	if !nord.saw["xlsx"] {
		nord.xlsxPath = "all-providers-results.xlsx"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if protocols.onIKE() {
		// Initialize the shared IKE materials before concurrent provider scans.
		prime, _ := new(big.Int).SetString(modp2048, 16)
		warmIKE(prime, 32)
	}
	names := append([]string{"nordvpn"}, gluetunProviders...)
	// Share the endpoint worker budget across active providers.
	workers := min(4, nord.conc)
	nord.conc /= workers
	gluetun.conc = nord.conc

	type outcome struct {
		scan providerScan
		err  error
	}
	results := parallel(names, workers, func(name string) outcome {
		if err := ctx.Err(); err != nil {
			return outcome{err: err}
		}
		if name == "nordvpn" {
			scan, err := scanNord(ctx, nord)
			return outcome{scan, err}
		}
		opts := gluetun
		opts.name = name
		scan, err := scanGluetun(ctx, opts)
		return outcome{scan, err}
	})

	combined := providerScan{source: "NordVPN and gluetun-servers"}
	var failures []error
	for i, result := range results {
		name := names[i]
		if result.err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, result.err))
			continue
		}
		if combined.fetched.IsZero() || result.scan.fetched.Before(combined.fetched) {
			combined.fetched = result.scan.fetched
		}
		for _, hit := range result.scan.hits {
			hit.Provider = name
			combined.hits = append(combined.hits, hit)
		}
	}
	if len(failures) == len(names) {
		return errors.Join(failures...)
	}
	if err := combined.write(nord.jsonPath, nord.csvPath, nord.xlsxPath, true); err != nil {
		failures = append(failures, err)
	}
	if ctx.Err() != nil {
		failures = append(failures, ctx.Err())
	}
	return errors.Join(failures...)
}
