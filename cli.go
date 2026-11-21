package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	os.Exit(runCLI(os.Args[1:]))
}

func runCLI(args []string) int {
	if len(args) == 0 {
		return runScanCLI(args)
	}
	if args[0] == "help" {
		if len(args) == 1 {
			return runScanCLI([]string{"-help"})
		}
		args = []string{args[1], "-help"}
		if args[0] == "scan" {
			return runScanCLI(args[1:])
		}
	}

	command, flags := args[0], args[1:]
	var err error
	switch command {
	case "profile":
		err = runProfileCLI(flags)
	case "accounts":
		err = runAccountsCLI(flags)
	case "config":
		err = runConfigCLI(flags)
	case "interactive", "-interactive":
		switch {
		case len(flags) == 0:
			err = newTerminalUI(os.Stdin, os.Stdout).run()
		case len(flags) == 1 && wantsHelp(flags):
			fmt.Println("Usage: vpnban interactive")
			fmt.Println("Choose a scan, create an Apple profile, manage accounts, or change the account directory.")
			fmt.Println("Use arrows and Enter in a terminal or enter the displayed option number.")
		default:
			err = errors.New("usage: vpnban interactive")
		}
	default:
		return runScanCLI(args)
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌", err)
		return 1
	}
	return 0
}

func commandFlags(name, usage, details string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: "+usage)
		fmt.Fprintln(flags.Output(), details)
		flags.PrintDefaults()
	}
	return flags
}

func wantsHelp(args []string) bool {
	return len(args) == 0 || args[0] == "-h" || args[0] == "-help" || args[0] == "--help"
}

func runProfileCLI(args []string) error {
	flags := commandFlags("profile", "vpnban profile -account ALIAS [options]",
		"Create an Apple IKEv2 .mobileconfig from a vpnban provider report.\n"+
			"Only servers for the account's provider that passed both IKEv2 phases are selected, fastest first.\n"+
			"Save an account with vpnban accounts add. Use the provider's manual IKEv2 service credentials.")
	alias := flags.String("account", "", "saved account alias")
	var opts profileOptions
	flags.StringVar(&opts.input, "input", "", "provider or all-providers JSON report (defaults to the provider report, then the combined report)")
	flags.StringVar(&opts.output, "output", "", "output .mobileconfig path (default: ALIAS.mobileconfig)")
	flags.StringVar(&opts.ca, "ca", "", "optional provider root CA in PEM or DER format")
	flags.StringVar(&opts.remoteID, "remote-id", "", "override the server certificate identity with the provider's IKEv2 remote ID")
	flags.IntVar(&opts.limit, "limit", defaultProfileLimit, "maximum number of VPN entries")
	if len(args) == 0 {
		flags.Usage()
		return nil
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *alias == "" || flags.NArg() != 0 {
		return errors.New("profile requires -account ALIAS. Run vpnban help profile for options")
	}
	dir, err := accountDir()
	if err != nil {
		return err
	}
	account, err := loadAccount(dir, *alias)
	if err != nil {
		return err
	}
	path, count, err := makeAppleProfile(account, opts)
	if err == nil {
		fmt.Printf("Wrote %s with %d verified IKEv2 endpoints.\n", path, count)
	}
	return err
}

func runAccountsCLI(args []string) error {
	switch {
	case wantsHelp(args):
		fmt.Println("Usage: vpnban accounts list")
		fmt.Println("       vpnban accounts add -name ALIAS -provider PROVIDER [-username USERNAME]")
		fmt.Println("Use a unique alias for each VPN account. Adding an existing alias updates that account.")
		fmt.Println("Passwords are read from the terminal without echo or from standard input.")
		return nil
	case args[0] == "list" && len(args) == 1:
		dir, err := accountDir()
		if err != nil {
			return err
		}
		accounts, err := listAccounts(dir)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			fmt.Println("No accounts saved.")
		}
		for _, account := range accounts {
			fmt.Printf("%s\t%s\n", account.Alias, account.Provider)
		}
		return nil
	case args[0] == "add":
		return addAccountCLI(args[1:])
	default:
		return errors.New("run vpnban help accounts for account commands")
	}
}

func addAccountCLI(args []string) error {
	flags := commandFlags("accounts add", "vpnban accounts add -name ALIAS -provider PROVIDER [options]",
		"Save an account for Apple profiles. The password is read from standard input.")
	alias := flags.String("name", "", "unique alias, for example nord-work")
	provider := flags.String("provider", "", "nordvpn or a supported gluetun provider")
	user := flags.String("username", "", "service username (prompted if omitted)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !accountAlias.MatchString(*alias) || flags.NArg() != 0 {
		return errors.New("accounts add requires -name with lowercase letters, digits or hyphens")
	}
	canonical, err := canonicalAccountProvider(*provider)
	if err != nil {
		return err
	}
	ui := newTerminalUI(os.Stdin, os.Stdout)
	if *user == "" {
		*user, err = ui.ask("Service username", "")
		if err != nil {
			return err
		}
	}
	password, err := ui.password()
	if err != nil {
		return err
	}
	dir, err := accountDir()
	if err != nil {
		return err
	}
	if err := saveAccount(dir, vpnAccount{Alias: *alias, Provider: canonical, Username: *user, Password: password}); err != nil {
		return err
	}
	fmt.Printf("Saved %s (%s).\n", *alias, canonical)
	return nil
}

func runConfigCLI(args []string) error {
	var dir string
	var err error
	switch {
	case wantsHelp(args):
		fmt.Println("Usage: vpnban config path")
		fmt.Println("       vpnban config set-dir DIRECTORY")
		fmt.Println("Changes the active directory for saved accounts. VPNBAN_CONFIG_DIR overrides it for a run.")
		return nil
	case args[0] == "path" && len(args) == 1:
		dir, err = accountDir()
	case args[0] == "set-dir" && len(args) == 2:
		dir, err = setAccountDir(args[1])
	default:
		return errors.New("run vpnban help config for settings commands")
	}
	if err == nil {
		fmt.Println(dir)
	}
	return err
}
