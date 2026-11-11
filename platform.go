package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// OpenWrt routers have limited memory and connection tracking capacity.
func isOpenWrt() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := os.Stat("/etc/openwrt_release")
	return err == nil
}

func defaultWorkers(standard int) int {
	if !isOpenWrt() {
		return standard
	}
	n := runtime.NumCPU() * 2
	if n < 4 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	return n
}

// Use RAM-backed storage on OpenWrt to avoid repeated writes to router flash.
func cacheBase() (string, error) {
	if isOpenWrt() {
		return filepath.Join(os.TempDir(), "vpnban-cache"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "nordike"), nil
}
