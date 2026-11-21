package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var errMenuBack = errors.New("back")

type terminalUI struct {
	input  *os.File
	reader *bufio.Reader
	output *os.File
	screen bool
	color  bool
}

func newTerminalUI(input, output *os.File) terminalUI {
	terminal := os.Getenv("TERM")
	screen := terminal != "" && terminal != "dumb" &&
		term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd()))
	return terminalUI{input: input, reader: bufio.NewReader(input), output: output,
		screen: screen, color: screen && os.Getenv("NO_COLOR") == ""}
}

func (u terminalUI) ask(label, fallback string) (string, error) {
	fmt.Fprint(u.output, label)
	if fallback != "" {
		fmt.Fprintf(u.output, " [%s]", fallback)
	}
	fmt.Fprint(u.output, ": ")
	line, err := u.readLine()
	return or(strings.TrimSpace(line), fallback), err
}

func (u terminalUI) readLine() (string, error) {
	line, err := u.reader.ReadString('\n')
	if errors.Is(err, io.EOF) && line != "" {
		err = nil
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), err
}

func (u terminalUI) password() (string, error) {
	if !term.IsTerminal(int(u.input.Fd())) {
		return u.readLine()
	}
	fmt.Fprint(u.output, "Service password: ")
	password, err := term.ReadPassword(int(u.input.Fd()))
	fmt.Fprintln(u.output)
	return string(password), err
}

func (u terminalUI) menu(title string, options []string) (int, error) {
	if u.screen {
		width, height, err := term.GetSize(int(u.output.Fd()))
		if err == nil && width >= 34 && height >= 9 {
			if state, err := term.MakeRaw(int(u.input.Fd())); err == nil {
				defer term.Restore(int(u.input.Fd()), state)
				return u.screenMenu(title, options, height-7)
			}
		}
	}
	fmt.Fprintf(u.output, "\nVPNBAN  /  %s\n\n", title)
	for i, label := range options {
		fmt.Fprintf(u.output, "  %2d  %s\n", i+1, label)
	}
	for {
		value, err := u.ask("Select a number (0 to go back)", "")
		if err != nil {
			return 0, err
		}
		if value == "0" || value == "q" {
			return 0, errMenuBack
		}
		if selected, err := strconv.Atoi(value); err == nil && selected > 0 && selected <= len(options) {
			return selected - 1, nil
		}
		fmt.Fprintln(u.output, "Choose one of the displayed numbers.")
	}
}

func (u terminalUI) screenMenu(title string, options []string, visible int) (int, error) {
	fmt.Fprint(u.output, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(u.output, "\x1b[?25h\x1b[?1049l")
	selected, start := 0, 0
	for {
		if selected < start {
			start = selected
		} else if selected >= start+visible {
			start = selected - visible + 1
		}
		fmt.Fprint(u.output, "\x1b[H\x1b[2J")
		fmt.Fprint(u.output, u.style("  ◆ VPNBAN", "1;96"), "\r\n")
		fmt.Fprint(u.output, u.style("  "+title, "1"), "\r\n\r\n")
		for i := start; i < min(len(options), start+visible); i++ {
			if i == selected {
				fmt.Fprint(u.output, u.style("  ›  "+options[i], "1;96"), "\r\n")
			} else {
				fmt.Fprint(u.output, "     ", options[i], "\r\n")
			}
		}
		fmt.Fprint(u.output, "\r\n", u.style("  ↑ ↓ Move   Enter Select   q Back", "2"), "\r\n")
		key, err := u.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if key == '\x1b' {
			if prefix, err := u.reader.ReadByte(); err == nil && (prefix == '[' || prefix == 'O') {
				key, err = u.reader.ReadByte()
				if err != nil {
					return 0, err
				}
			}
		}
		switch key {
		case 'A', 'k':
			selected = (selected + len(options) - 1) % len(options)
		case 'B', 'j':
			selected = (selected + 1) % len(options)
		case '\r', '\n':
			return selected, nil
		case 'q', 3, 4:
			return 0, errMenuBack
		}
	}
}

func (u terminalUI) style(s, codes string) string {
	if u.color {
		return "\x1b[" + codes + "m" + s + "\x1b[0m"
	}
	return s
}

func (u terminalUI) run() error {
	actions := []string{"Check VPN providers", "Check from a file", "Build Apple profile", "Accounts", "Settings", "Exit"}
	for {
		n, err := u.menu("Main menu", actions)
		if errors.Is(err, errMenuBack) || errors.Is(err, io.EOF) || (err == nil && n == len(actions)-1) {
			return nil
		}
		if err != nil {
			return err
		}
		switch n {
		case 0:
			err = u.providerWizard()
		case 1:
			err = u.fileWizard()
		case 2:
			err = u.profileWizard()
		case 3:
			err = u.accountsWizard()
		case 4:
			err = u.settingsWizard()
		}
		if errors.Is(err, errMenuBack) {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			fmt.Fprintln(u.output, u.style("Error", "1;31"), err)
		}
		if _, err := u.ask("Press Enter to return to the menu", ""); err != nil {
			return nil
		}
	}
}

func (u terminalUI) chooseProvider(allowAll bool) (string, error) {
	providers := append([]string{"nordvpn"}, gluetunProviders...)
	labels := append([]string{"NordVPN"}, gluetunProviders...)
	if allowAll {
		providers = append([]string{""}, providers...)
		labels = append([]string{"All providers"}, labels...)
	}
	n, err := u.menu("VPN provider", labels)
	return providers[n], err
}

func (u terminalUI) providerWizard() error {
	provider, err := u.chooseProvider(true)
	if err != nil {
		return err
	}
	n, err := u.menu("VPN protocols", []string{"Provider default", "IKEv2", "WireGuard", "OpenVPN", "IKEv2, WireGuard and OpenVPN"})
	if err != nil {
		return err
	}
	args := []string{"-provider", provider}
	switch provider {
	case "":
		args = []string{"-all-providers"}
	case "nordvpn":
		args = []string{"-nord"}
	}
	if proto := []string{"", "ikev2", "wire", "openvpn", "ikev2,wire,openvpn"}[n]; proto != "" {
		args = append(args, "-proto", proto)
	}
	country, err := u.ask("Country filter (optional)", "")
	if err != nil {
		return err
	}
	if country != "" {
		args = append(args, "-country", country)
	}
	limit, err := u.ask("Endpoint limit per provider (0 for all)", "10")
	if err != nil {
		return err
	}
	if status := runScanCLI(append(args, "-limit", limit)); status != 0 {
		return fmt.Errorf("scan exited with status %d", status)
	}
	return nil
}

func (u terminalUI) fileWizard() error {
	path, err := u.ask("Host list or JSON report", "")
	if err != nil {
		return err
	}
	n, err := u.menu("Scan phase", []string{"Automatic", "DNS discovery only", "Probe saved addresses", "Discover and probe"})
	if err != nil {
		return err
	}
	if status := runScanCLI([]string{"-file", path, "-phase", []string{"", "1", "2", "all"}[n]}); status != 0 {
		return fmt.Errorf("scan exited with status %d", status)
	}
	return nil
}

func (u terminalUI) profileWizard() error {
	dir, err := accountDir()
	if err != nil {
		return err
	}
	accounts, err := listAccounts(dir)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return errors.New("save a VPN account under Accounts first")
	}
	labels := make([]string, len(accounts))
	for i, account := range accounts {
		labels[i] = account.Alias + "  ·  " + account.Provider
	}
	n, err := u.menu("Profile account", labels)
	if err != nil {
		return err
	}
	account, err := loadAccount(dir, accounts[n].Alias)
	if err != nil {
		return err
	}
	input, err := u.ask("Provider or combined JSON report", defaultReport(account.Provider))
	if err != nil {
		return err
	}
	output, err := u.ask("Apple profile output", account.Alias+".mobileconfig")
	if err != nil {
		return err
	}
	ca, err := u.ask("Provider root certificate (optional)", "")
	if err != nil {
		return err
	}
	path, count, err := makeAppleProfile(account, profileOptions{input: input, output: output, ca: ca, limit: defaultProfileLimit})
	if err == nil {
		fmt.Fprintf(u.output, "Wrote %s with %d verified IKEv2 endpoints.\n", path, count)
	}
	return err
}

func (u terminalUI) accountsWizard() error {
	dir, err := accountDir()
	if err != nil {
		return err
	}
	n, err := u.menu("Accounts", []string{"Show saved accounts", "Add or update an account"})
	if err != nil {
		return err
	}
	if n == 0 {
		accounts, err := listAccounts(dir)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			fmt.Fprintln(u.output, "No accounts saved.")
		}
		for _, account := range accounts {
			fmt.Fprintf(u.output, "  %-20s %s\n", account.Alias, account.Provider)
		}
		return nil
	}
	provider, err := u.chooseProvider(false)
	if err != nil {
		return err
	}
	alias, err := u.ask("Account alias", "")
	if err != nil {
		return err
	}
	if _, err := accountPath(dir, alias); err != nil {
		return err
	}
	user, err := u.ask("Service username", "")
	if err != nil {
		return err
	}
	password, err := u.password()
	if err != nil {
		return err
	}
	if err := saveAccount(dir, vpnAccount{Alias: alias, Provider: provider, Username: user, Password: password}); err != nil {
		return err
	}
	fmt.Fprintf(u.output, "Saved %s (%s).\n", alias, provider)
	return nil
}

func (u terminalUI) settingsWizard() error {
	current, err := accountDir()
	if err != nil {
		return err
	}
	fmt.Fprintf(u.output, "\nAccount directory: %s\n", current)
	dir, err := u.ask("Use another directory (blank keeps the current directory)", "")
	if err != nil || dir == "" {
		return err
	}
	dir, err = setAccountDir(dir)
	if err == nil {
		fmt.Fprintf(u.output, "Account directory set to %s.\n", dir)
	}
	return err
}
