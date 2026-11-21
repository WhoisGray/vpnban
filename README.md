<div align="center">

<img src="docs/assets/vpnban-banner.svg" alt="vpnban — the keeper of VPN checks" width="100%">

# vpnban · بان

**The keeper of VPN checks. Choose your protocol. Keep the results.**

[English](README.md) · [فارسی](README.fa.md)

[![CI](https://github.com/WhoisGray/vpnban/actions/workflows/ci.yml/badge.svg)](https://github.com/WhoisGray/vpnban/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/WhoisGray/vpnban?color=38bdf8)](https://github.com/WhoisGray/vpnban/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Contributions welcome](https://img.shields.io/badge/contributions-welcome-34d399)](#contribute)

[Get started](#quick-start) · [Usage](#usage) · [Provider roadmap](#provider-roadmap) · [Contribute](#contribute)

</div>

---

**vpnban** (بان, “the keeper”) is a Go command-line tool that keeps VPN server checks in mind. Discover NordVPN servers through its official API, use gluetun-servers JSON for other providers, or bring your own host list. Export checks as JSON, CSV, and Excel.

> Checks show what responds from your current network. They do not guarantee successful VPN login, a working tunnel, or browsing performance. Only scan servers you have permission to check.

## Why vpnban?

- **Choose the checks:** IKEv2 by default; WireGuard, OpenVPN, combinations, or a full NordVPN scan.
- **Narrow the search:** filter NordVPN servers by country and load, then limit how many to check.
- **Bring your own list:** use domain/IPv4 text files, supported JSON catalogs, or previous results.
- **Keep useful reports:** JSON for automation, CSV for analysis, XLSX for sharing.
- **Run across platforms:** macOS, Linux, and Windows releases for amd64 and arm64.
- **Reuse discovery:** separate six-hour caches for NordVPN and each gluetun-servers provider, with an explicit refresh option.

## Quick start

### 1. Install

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/WhoisGray/vpnban/refs/heads/main/install.sh | sh
```

**Windows · PowerShell**

```powershell
irm https://raw.githubusercontent.com/WhoisGray/vpnban/refs/heads/main/install.ps1 | iex
```

The installers select your platform and architecture and verify the archive against the release SHA256 checksums. If the installer asks you to add a directory to `PATH`, follow its printed instruction and reopen your terminal. You can inspect [install.sh](install.sh) or [install.ps1](install.ps1) before running it.

Prefer a manual install? Download an archive and `checksums.txt` from [Releases](https://github.com/WhoisGray/vpnban/releases/latest).

| System | amd64 / Intel | arm64 / Apple silicon |
| --- | --- | --- |
| macOS | `darwin_amd64.tar.gz` | `darwin_arm64.tar.gz` |
| Linux | `linux_amd64.tar.gz` | `linux_arm64.tar.gz` |
| Windows | `windows_amd64.zip` | `windows_arm64.zip` |

Archive names begin with `vpnban_<version>_`. Verify the downloaded archive before extraction: use `shasum -a 256 <archive>` on macOS, `sha256sum <archive>` on Linux, or `Get-FileHash <archive> -Algorithm SHA256` in PowerShell, and compare with the matching line in `checksums.txt`. Extract the binary into a directory on your `PATH`; on Unix, make it executable with `chmod +x vpnban`. macOS binaries are unsigned; if Gatekeeper blocks a verified download, remove its quarantine attribute with `xattr -d com.apple.quarantine /path/to/vpnban`.

**With Go 1.22+**

```bash
go install github.com/whoisGray/vpnban@latest
```

### 2. Run a focused check

```bash
vpnban -version
vpnban -nord -country se -limit 10
```

This checks IKEv2 on up to ten Swedish NordVPN servers. In PowerShell, use `.\vpnban.exe` if the executable is in your current folder rather than on `PATH`.

### 3. Open the reports

NordVPN mode writes `nord-results.json`, `nord-results.csv`, and `nord-results.xlsx` in the current directory. Custom-file mode uses the `vpnban-results` prefix. Override paths with `-json`, `-csv`, and `-xlsx`.

## Usage

### NordVPN discovery and protocol selection

```bash
vpnban -nord                              # IKEv2 only
vpnban -nord -proto wire                   # WireGuard only
vpnban -nord -proto openvpn                # OpenVPN UDP + TCP
vpnban -nord -proto ikev2,wire              # Selected protocols
vpnban -nord -proto full                   # All checks + firewall diagnostics
vpnban -nord -country se,de -limit 40 -c 32
vpnban -nord -refresh                      # Bypass the cached server list
```

`-proto` applies to NordVPN mode. It accepts `ikev2`, `wire` (aliases: `wg`, `wireguard`), `openvpn`, comma-separated combinations, or `full`. Servers without any selected protocol are filtered out. Full mode also probes UDP/53 and TCP/80 to help identify unusual firewall responses.

### Your own server list

Create `servers.txt` with one authorized domain or IPv4 address per line. Blank lines and `#` comments are supported.

```bash
vpnban -file servers.txt
vpnban -file servers.json
vpnban -file servers.txt -phase 1           # DNS discovery only
vpnban -file vpnban-results.json -phase 2     # Probe saved addresses
```

Custom-file mode uses its own discovery and probing flow; the NordVPN `-proto` selector does not configure it. See [main.go](main.go) for supported JSON catalog and saved-result formats.

### Other providers via gluetun-servers

```bash
vpnban -provider mullvad -proto wire -country se -limit 40
vpnban -provider protonvpn -proto openvpn -country Germany -limit 20
vpnban -provider "vpn unlimited" -proto ikev2 -limit 10
vpnban -provider surfshark -proto wire,openvpn -refresh
```

Supported names: `airvpn`, `cyberghost`, `expressvpn`, `fastestvpn`, `giganews`, `hidemyass`, `ipvanish`, `ivpn`, `mullvad`, `ovpn`, `privado`, `private internet access`, `privatevpn`, `protonvpn`, `purevpn`, `slickvpn`, `surfshark`, `torguard`, `vpn unlimited`, `vpnsecure`, `vyprvpn`, `windscribe`.

This mode downloads the matching JSON from [gluetun-servers](https://github.com/qdm12/gluetun-servers/tree/main/pkg/servers). `-country` accepts a country name or ISO two-letter code where the catalog includes a country. `-limit` caps the number of IP endpoints probed. `-proto` uses the JSON's `vpn`, `tcp`, `udp`, and `wgpubkey` fields to select supported OpenVPN and WireGuard checks. IKEv2 is a discovery probe because these JSON files do not declare IKEv2 support. For IKEv2, the Remote ID defaults to the listed hostname; **VPN Unlimited always uses `ironnodes.com`** even though its gateway hostnames end in `vpnunlimitedapp.com`. Its transport hostname is otherwise used normally. An explicit `-ike-id` can override the default for other providers.

Provider reports use `<provider>-results.json`, `.csv`, and `.xlsx` by default. Each provider has a separate cache under `~/.cache/nordike/gluetun`, using `-ttl` and `-refresh` without changing NordVPN's cache.

### Check all supported providers

```bash
vpnban -all-providers -country se -limit 10 -c 32
vpnban -all-providers -proto ikev2 -limit 10
```

`-all-providers` checks NordVPN and all supported gluetun providers in one run. It defaults to IKEv2, WireGuard, and OpenVPN. `-limit` applies to each provider, and `-c` is shared across providers. The combined `all-providers-results.json`, `.csv`, and `.xlsx` reports include a `provider` column. If a provider fails, vpnban writes the results it obtained, reports the failure, and exits with a nonzero status.

### Interactive terminal menu

```bash
vpnban interactive
```

Use the arrow keys and Enter to choose a provider scan, check a host list, build an Apple profile, manage accounts, or open settings. Use the numbered menu when input or output is redirected. The scan wizard starts with a limit of 10 endpoints per provider.

### Build an Apple IKEv2 profile

1. Save each VPN account under a different alias. You can save multiple accounts from the same provider. The terminal hides the service password as you type.

   ```bash
   vpnban accounts add -name nord-home -provider nordvpn
   vpnban accounts add -name nord-work -provider nordvpn
   vpnban accounts list
   ```

2. Run an IKEv2 check and build the profile from its JSON report.

   ```bash
   vpnban -nord -proto ikev2 -limit 30
   vpnban profile -account nord-home -input nord-results.json -output nord-home.mobileconfig
   ```

   You can also use `all-providers-results.json` as input. To generate it for this workflow, run `vpnban -all-providers -proto ikev2 -limit 30`. Each profile uses one account and includes only its provider's servers that passed both IKEv2 phases, fastest first. The default is eight VPN entries. Use `-limit` to change it.

3. On macOS, run `plutil -lint nord-home.mobileconfig` to check the file format. To install on iPhone or iPad, open the file and go to **Settings → General → VPN & Device Management**. On Mac, open the file and use **System Settings → General → Device Management**. [Apple installation guide for Mac](https://support.apple.com/guide/mac-help/configuration-profiles-standardize-settings-mh35561/mac)

This works with any listed provider that offers manual IKEv2 with username and password authentication and has a verified server in the report. Obtain the service credentials and certificate settings from your provider. Use `-ca path/to/provider-root.pem` to include its root CA or follow its separate trust setup instructions. Use `-remote-id` if the provider specifies a different certificate identity. NordVPN documents its service credentials and root CA setup. [NordVPN manual IKEv2 guide](https://support.nordvpn.com/hc/en-us/articles/20371657885329-How-to-connect-to-NordVPN-with-a-manual-IKEv2-IPSec-connection-on-an-iOS-device)

Saved accounts and exported profiles contain service credentials. Keep those files private. The account list shows aliases and providers only.

### Account storage and command help

```bash
vpnban config path
vpnban config set-dir "$HOME/vpnban-accounts"
vpnban help profile
vpnban help accounts
vpnban help config
```

vpnban uses your OS user configuration directory by default. **Settings** in the interactive menu changes the same location as `config set-dir`. Selecting a new directory copies your current accounts there and keeps the original files. Selecting an existing directory opens its accounts. Set `VPNBAN_CONFIG_DIR` to use a directory for a single run.

### Read the results

Servers that pass both IKEv2 phases appear first, in ascending latency order. Provider reports rank those servers by IKEv2 RTT. File-mode reports use average TCP RTT when available and fall back to IKEv2 RTT.

| Value | Meaning |
| --- | --- |
| `real` | Response matched the expected protocol checks |
| `p1` | Only the first IKEv2 phase succeeded |
| `silent` | No response was received |
| `fresh=yes` | The scorer accepted protocol evidence; this can include IKEv2 phase-one-only evidence |
| `unproven` | No conclusive protocol evidence |

A silent server may be filtered, unreachable, or simply slow. Treat a result as evidence from one network at one point in time.

### Cache and DNS troubleshooting

NordVPN discovery is cached under `~/.cache/nordike` for six hours by default. Use `-refresh` to fetch again or `-ttl` to change the lifetime.

In custom-file discovery, if DNS returns only private addresses, vpnban stops by default. Check your VPN and resolver settings before retrying. `-allow-poisoned` overrides this safeguard when that behavior is intentional.

## Options

| Flag | Default | Purpose |
| --- | --- | --- |
| `-nord` | `false` | Discover and check NordVPN servers |
| `-provider` | — | Discover another provider from gluetun-servers JSON |
| `-all-providers` | `false` | Check every supported provider and combine results |
| `-proto` | `ikev2`; all-provider mode uses `ikev2,wire,openvpn` | Provider protocol selection |
| `-country` | All | Comma-separated country codes or names |
| `-limit` | `0` | Server or endpoint limit; zero means all (NordVPN: lowest load first) |
| `-load` | `100` | Maximum NordVPN server load |
| `-offline` | `false` | Include NordVPN servers not marked online |
| `-refresh` | `false` | Download the selected provider catalog again |
| `-ttl` | `6h` | Selected provider cache lifetime |
| `-c` | `80`; provider modes: `32` | Concurrent workers |
| `-ike-timeout` | `1s` | IKEv2 timeout |
| `-tcp-timeout` | `1200ms`; NordVPN: `1s` | TCP timeout |
| `-ike-user` | Mode-dependent | IKE_AUTH user identity |
| `-ike-id` | Mode-dependent | IKE_AUTH remote identity |
| `-file` | — | Input catalog, host list, or saved results |
| `-phase` | Automatic | `1` discovery, `2` probing, `all` both |
| `-dns` | `1.1.1.1:53` | Discovery DNS resolver |
| `-allow-poisoned` | `false` | Continue despite private DNS answers |
| `-json`, `-csv`, `-xlsx` | Mode-dependent | Report paths |
| `-version` | — | Print version and exit |
| `-update` | — | Replace this binary with the latest GitHub release |

IKE identities default to `health@nordvpn.com` / `nordvpn.com` in NordVPN mode, and `this is health check` / `ironnodes.com` in file mode.

To update, run `vpnban -update`. Older binaries such as v1.0.1 do not recognize this flag; rerun the installer instead.

## Provider roadmap

NordVPN uses its official API. The other providers above use gluetun-servers JSON. Custom lists are supported separately.

- [x] NordVPN — official API discovery, filters, cache, and protocol checks.
- [x] Custom domain / IPv4 lists and supported JSON input.
- [x] 22 other providers — gluetun-servers JSON discovery, filters, separate caches, and protocol checks.

Have another provider in mind? [Propose an integration](https://github.com/WhoisGray/vpnban/issues/new?title=Provider%20request%3A%20) with the provider name, a documented server source, available protocols, and any account requirements.

## Contribute

**Help make vpnban useful on more networks.** Provider research, Go code, tests, documentation, Persian translations, and reproducible bug reports are welcome.

1. Open an issue to discuss a provider or pick an existing task; mention what you plan to work on.
2. Fork the repository and create a focused branch.
3. For a provider, document the server source, authentication needs, rate limits, supported protocols, and limitations. Never commit credentials or private keys.
4. Add fixture-based parsing tests and error cases; reuse the current probe code where appropriate. Mark unsupported checks clearly.
5. Update both READMEs, run the checks below, and open a pull request with sample output and validation notes.

Useful starting points: [nord.go](nord.go) for discovery and filtering, [main.go](main.go) for CLI/input handling, and [ike.go](ike.go), [wg.go](wg.go), [ovpn.go](ovpn.go) for protocol checks. A shared provider interface is still a roadmap item.

```bash
git clone https://github.com/WhoisGray/vpnban.git
cd vpnban
go test ./...
go build -o vpnban .
./vpnban -version
```

CI tests and cross-compiles six OS/architecture targets. Version tags trigger the release workflow, which tests, packages binaries, and publishes SHA256 checksums.

---

If vpnban helps you, **give it a star ⭐** and share your feedback. Every provider contribution makes the project more useful.
