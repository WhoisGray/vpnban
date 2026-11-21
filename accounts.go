package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type vpnAccount struct {
	Alias    string `json:"alias"`
	Provider string `json:"provider"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type localSettings struct {
	AccountDir string `json:"account_dir"`
}

var accountAlias = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)

func canonicalAccountProvider(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "nord", "nordvpn":
		return "nordvpn", nil
	default:
		return canonicalProvider(name)
	}
}

func settingsFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "vpnban", "settings.json"), nil
}

func accountDir() (string, error) {
	if override := os.Getenv("VPNBAN_CONFIG_DIR"); override != "" {
		return filepath.Abs(override)
	}
	path, err := settingsFile()
	if err != nil {
		return "", err
	}
	var settings localSettings
	if err := readLocalJSON(path, &settings); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if settings.AccountDir != "" {
		return settings.AccountDir, nil
	}
	return filepath.Join(filepath.Dir(path), "accounts"), nil
}

func setAccountDir(dir string) (string, error) {
	if os.Getenv("VPNBAN_CONFIG_DIR") != "" {
		return "", errors.New("unset VPNBAN_CONFIG_DIR to change the saved account directory")
	}
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("account directory is required")
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path, err := settingsFile()
	if err != nil {
		return "", err
	}
	current, err := accountDir()
	if err != nil {
		return "", err
	}
	if err := prepareAccountDir(current, absolute); err != nil {
		return "", err
	}
	return absolute, writeLocalJSON(path, localSettings{AccountDir: absolute})
}

func prepareAccountDir(source, destination string) error {
	info, err := os.Stat(destination)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", destination)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".vpnban-accounts-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := copyAccounts(source, staging); err != nil {
		return err
	}
	return os.Rename(staging, destination)
}

func copyAccounts(source, destination string) error {
	accounts, err := listAccounts(source)
	if err != nil {
		return err
	}
	for _, listed := range accounts {
		account, err := loadAccount(source, listed.Alias)
		if err != nil {
			return err
		}
		if err := saveAccount(destination, account); err != nil {
			return err
		}
	}
	return nil
}

func accountPath(dir, alias string) (string, error) {
	if !accountAlias.MatchString(alias) {
		return "", errors.New("use a 1–48 character account alias with lowercase letters, digits, and hyphens")
	}
	return filepath.Join(dir, "account-"+alias+".json"), nil
}

func saveAccount(dir string, account vpnAccount) error {
	path, err := accountPath(dir, account.Alias)
	if err != nil {
		return err
	}
	account.Provider, err = canonicalAccountProvider(account.Provider)
	if err != nil {
		return err
	}
	if strings.TrimSpace(account.Username) == "" || account.Password == "" {
		return errors.New("service username and password are required")
	}
	return writeLocalJSON(path, account)
}

func loadAccount(dir, alias string) (vpnAccount, error) {
	path, err := accountPath(dir, alias)
	if err != nil {
		return vpnAccount{}, err
	}
	var account vpnAccount
	if err := readLocalJSON(path, &account); err != nil {
		return vpnAccount{}, err
	}
	account.Provider, err = canonicalAccountProvider(account.Provider)
	if err != nil || account.Alias != alias || strings.TrimSpace(account.Username) == "" || account.Password == "" {
		return vpnAccount{}, fmt.Errorf("invalid account file for %q", alias)
	}
	return account, nil
}

type accountSummary struct {
	Alias    string
	Provider string
}

func listAccounts(dir string) ([]accountSummary, error) {
	files, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var accounts []accountSummary
	for _, file := range files {
		alias, ok := strings.CutPrefix(file.Name(), "account-")
		if file.IsDir() || !ok || !strings.HasSuffix(alias, ".json") {
			continue
		}
		alias = strings.TrimSuffix(alias, ".json")
		account, err := loadAccount(dir, alias)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, accountSummary{Alias: account.Alias, Provider: account.Provider})
	}
	return accounts, nil
}

func readLocalJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		// A parser error can quote credential data.
		return fmt.Errorf("invalid JSON in %s", path)
	}
	return nil
}

func writeLocalJSON(path string, data any) error {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writePrivateFile(path, append(payload, '\n'))
}

func writePrivateFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".vpnban-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
