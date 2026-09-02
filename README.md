# x-cmd

[![Latest Release](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![Build and Release](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

English | [简体中文](README.zh-CN.md)

`x-cmd` is a command-line wrapper and manager for xray-core. Every operation is available as a script-friendly command, while running it without arguments opens an interactive menu.

> **GitHub access is optional:** In regions without direct GitHub access, installation supports a GitHub mirror prefix. The installer saves this setting, and `x-cmd` reuses it for future xray-core downloads and application updates.
>
> The current GitHub mirror is `https://github.uzfdafw.cc`.

## Features

- Install, inspect, and switch xray-core versions
- Override GitHub download URLs for regions where GitHub is unavailable
- Manage multiple v2rayN subscriptions and standalone share links
- Perform real proxy connection tests and optionally remove failed nodes
- Run an HTTP/SOCKS mixed proxy on `127.0.0.1:1091`
- Start and stop xray, inspect its status, and control the system proxy
- Check GitHub Releases and update the current executable online
- Build Windows, Linux, and macOS release artifacts on multiple architectures

## Installation

### Linux and macOS

```sh
curl -fsSLO https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh
sh install.sh
```

Install a specific version/location or use a GitHub mirror:

```sh
sh install.sh --version v0.4.1 --install-dir "$HOME/.local/bin"
sh install.sh --github-mirror github.uzfdafw.cc
```

The mirror address is prepended to the complete GitHub URL:

```text
https://github.com/accloud-proj/x-cmd/releases/latest/download/...
https://github.uzfdafw.cc/https://github.com/accloud-proj/x-cmd/releases/latest/download/...
```

If GitHub itself is unreachable, fetch the installer from the mirrored Release URL first:

```sh
MIRROR="https://github.uzfdafw.cc"
curl -fsSL "$MIRROR/https://github.com/accloud-proj/x-cmd/releases/latest/download/install.sh" -o install.sh
sh install.sh --github-mirror "$MIRROR"
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1
```

Optional parameters:

```powershell
.\install.ps1 -Version v0.4.1 -InstallDir "$env:LOCALAPPDATA\x-cmd\bin"
.\install.ps1 -GitHubMirror github.uzfdafw.cc
```

To obtain the PowerShell installer through the mirror:

```powershell
$mirror = "https://github.uzfdafw.cc"
Invoke-WebRequest "$mirror/https://github.com/accloud-proj/x-cmd/releases/latest/download/install.ps1" -OutFile install.ps1
.\install.ps1 -GitHubMirror $mirror
```

Both installers verify the archive with the SHA-256 value in the Release's `checksums.txt` before installation.

### Build from source

Go 1.27 or later is required.

```sh
go build -o x-cmd .
```

Run `x-cmd` without arguments for the interactive menu. The source version is stored in `version.Version`; tagged builds inject the tag version with `-ldflags`.

## Compatibility

### Node protocols

| Protocol          | Accepted link                         | Credentials             | Transport and security                                 |
| ----------------- | ------------------------------------- | ----------------------- | ------------------------------------------------------ |
| VMess             | `vmess://` Base64-encoded v2rayN JSON | UUID, alterId, security | TCP, WebSocket, gRPC, HTTP/2; none or TLS              |
| VLESS             | `vless://` URI                        | UUID, encryption, flow  | TCP, WebSocket, gRPC, HTTP/2; none, TLS, or REALITY    |
| Trojan            | `trojan://` URI                       | Password, optional flow | TCP, WebSocket, gRPC, HTTP/2; none, TLS, or REALITY    |
| Shadowsocks       | `ss://` SIP002 URI                    | Method and password     | Xray-supported AEAD methods; plugins are not supported |
| HTTP/HTTPS        | `http://` or `https://` URI           | Optional user/password  | HTTP upstream; HTTPS enables TLS to the upstream       |
| SOCKS5            | `socks://` or `socks5://` URI         | Optional user/password  | Xray SOCKS5 outbound                                   |
| Any Xray outbound | `xray://` Base64URL JSON              | Defined by JSON         | Native outbound object, passed to xray without loss    |

Where applicable, WebSocket host/path, gRPC service name, HTTP/2 host/path, TLS SNI, and REALITY `fp`, `pbk`, `sid`, and `spx` parameters are translated to xray configuration.

The native `xray://` form covers every xray-core outbound protocol, including WireGuard, Hysteria, Freedom, DNS, Blackhole, and Loopback, and remains compatible with future protocols. Encode one complete Xray `OutboundObject` as unpadded Base64URL and optionally append `#name`:

```text
xray://eyJwcm90b2NvbCI6IndpcmVndWFyZCIsInNldHRpbmdzIjp7Li4ufX0#my-wireguard
```

`protocol` and `settings` are required. The object is passed through unchanged, so advanced users may include `streamSettings`, `mux`, and other outbound fields. Blackhole, DNS, Freedom, and Loopback are Xray routing outbounds rather than remote proxy servers; they are supported by this format but are generally not useful as selectable subscription nodes.

There is no single interoperable v2rayN share URI for all WireGuard and Xray Hysteria configurations. Their complete settings should therefore use `xray://` instead of a guessed or lossy URI mapping. TUIC is not an xray-core outbound protocol. Shadowsocks SIP008 JSON/plugins and arbitrary full Xray configuration files are not accepted; `xray://` contains one outbound object only.

### Subscription formats

Only the v2rayN subscription format is accepted:

1. A Base64-encoded text document containing one share URI per line.
2. A plain-text document containing share URIs separated by whitespace.
3. Each URI must use VMess, VLESS, Trojan, Shadowsocks, HTTP(S), SOCKS5, or native `xray://` as described above.

Clash YAML, Clash.Meta/Mihomo YAML, sing-box JSON, Surge profiles, SIP008 JSON, and nested subscriptions are intentionally not parsed. Invalid lines are skipped; an update fails if no supported node remains.

## Core Management

```sh
x-cmd core show
x-cmd core install --version v25.8.3
x-cmd core install --version v25.8.3 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
```

## Subscription and Node Management

```sh
x-cmd sub add --name "Provider A" --url "https://example.com/subscription"
x-cmd sub list
x-cmd sub edit <ID> --name "New name" --url "https://example.com/new"
x-cmd sub update <ID>
x-cmd sub update all
x-cmd sub nodes <ID>
x-cmd sub delete <ID>

x-cmd node add --uri "vless://..."
x-cmd node list
x-cmd node list --subscription <SUBSCRIPTION_ID>
x-cmd node use <NODE_ID>
x-cmd node delete <NODE_ID>
```

An unambiguous ID prefix may be used. Subscription updates replace only nodes owned by that subscription and preserve standalone nodes. The first available node is selected automatically.

## Connection Testing

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <ID> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

This is not a server port check. `x-cmd` starts a temporary xray process for each node and sends an HTTP request through its local SOCKS5 inbound, validating the complete proxy path.

## Running the Proxy

```sh
x-cmd start
x-cmd status
x-cmd stop
x-cmd config set --listen-port 1091

x-cmd proxy enable
x-cmd proxy status
x-cmd proxy disable
```

`start` uses the active node and exposes an HTTP/SOCKS mixed inbound at `127.0.0.1:1091` by default. Restart after selecting another node. System proxy control updates Windows user Internet Settings, enabled macOS network services, or GNOME `gsettings` on Linux. It is not a transparent proxy or TUN.

## Updating x-cmd

```sh
x-cmd update check
x-cmd update install
```

The updater downloads the current platform artifact from the latest `accloud-proj/x-cmd` Release and replaces the executable. Its directory must be writable. Windows retains the previous executable as `x-cmd.exe.old`.

## GitHub Mirror Management

```sh
x-cmd github-mirror show
x-cmd github-mirror set github.uzfdafw.cc
x-cmd github-mirror delete
```

`set` replaces the current mirror, while `delete` restores direct GitHub access. `x-cmd config show` and `x-cmd config set --github-mirror URL` remain available.

## Release Builds

[The release workflow](.github/workflows/release.yml) builds Windows amd64/arm64/386, Linux amd64/arm64/armv7, and macOS amd64/arm64. Pushing a tag such as `v0.4.1` creates a Release with archives and `checksums.txt`.

## Data Directory

The GitHub mirror is stored with the other settings in the `settings.github_mirror` field of `config.json`. Default paths are:

- Windows: `%AppData%\x-cmd\config.json`
- Linux: `$XDG_CONFIG_HOME/x-cmd/config.json`, or `~/.config/x-cmd/config.json` when `XDG_CONFIG_HOME` is unset
- macOS: `~/Library/Application Support/x-cmd/config.json`

Override the complete configuration file path with `X_CMD_CONFIG`. Subscription URLs may contain credentials; do not publish this file.

## Uninstall

```sh
x-cmd uninstall --yes
```

This command disables the system proxy, stops xray, and removes `x-cmd` itself, update backups, the configuration file, and runtime data. Stored subscriptions and nodes are permanently deleted as well. When using the interactive menu, select `u. Uninstall` and confirm.

## License

Licensed under the [BSD 3-Clause License](LICENSE).
