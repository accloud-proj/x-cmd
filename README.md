# x-cmd

[![Latest Release](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![Build and Release](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

English | [简体中文](README.zh-CN.md)

`x-cmd` is a command-line wrapper and manager for xray-core. Every operation is available as a script-friendly command, while running it without arguments opens an interactive menu.

> **GitHub access is optional:** The installers and client try GitHub first, then switch to the built-in mirror when direct access fails. The selected fallback is saved and used for subsequent GitHub requests until it is manually changed or deleted.

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
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh)
```

The installer downloads the latest Release by default. To use a GitHub mirror:

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh) --github-mirror https://your-mirror.example
```

If GitHub is unreachable, open [scripts/install.sh](scripts/install.sh) in any environment that can access the repository, paste its complete contents into a local `install.sh`, and run:

```sh
bash install.sh
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1
```

Optional parameters:

```powershell
.\install.ps1 -GitHubMirror https://your-mirror.example
```

If GitHub is unreachable, open [scripts/install.ps1](scripts/install.ps1) in any environment that can access the repository, paste its complete contents into a local `install.ps1`, and run:

```powershell
.\install.ps1
```

## Compatibility

| Protocol          | Accepted link                         | Credentials             | Transport and security                                 |
| ----------------- | ------------------------------------- | ----------------------- | ------------------------------------------------------ |
| VMess             | `vmess://` Base64-encoded v2rayN JSON | UUID, alterId, security | TCP, WebSocket, gRPC, HTTP/2; none or TLS              |
| VLESS             | `vless://` URI                        | UUID, encryption, flow  | TCP, WebSocket, gRPC, HTTP/2; none, TLS, or REALITY    |
| Trojan            | `trojan://` URI                       | Password, optional flow | TCP, WebSocket, gRPC, HTTP/2; none, TLS, or REALITY    |
| Shadowsocks       | `ss://` SIP002 URI                    | Method and password     | Xray-supported AEAD methods; plugins are not supported |
| HTTP/HTTPS        | `http://` or `https://` URI           | Optional user/password  | HTTP upstream; HTTPS enables TLS to the upstream       |
| SOCKS5            | `socks://` or `socks5://` URI         | Optional user/password  | Xray SOCKS5 outbound                                   |
| Any Xray outbound | `xray://` Base64URL JSON              | Defined by JSON         | Native outbound object, passed to xray without loss    |

Supports Base64-encoded or plain-text v2rayN subscriptions. Supported node protocols are listed in the table above.

## Shell Completion

```sh
x-cmd completion install
x-cmd completion uninstall
```

The current shell is detected automatically. Bash, Zsh, Fish, and PowerShell can also be selected explicitly, for example `x-cmd completion install powershell`. Reopen the terminal after installation.

## Core Management

```sh
x-cmd core show
x-cmd core releases
x-cmd core install --version v26.3.27
x-cmd core install --version v26.3.27 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
```

## Subscription and Node Management

```sh
x-cmd sub add --name "Provider A" --url "https://example.com/subscription"
x-cmd sub list
x-cmd sub edit <NUMBER_OR_NAME> --name "New name" --url "https://example.com/new"
x-cmd sub update <NUMBER_OR_NAME>
x-cmd sub update all
x-cmd sub nodes <NUMBER_OR_NAME>
x-cmd sub delete <NUMBER_OR_NAME>

x-cmd node add --uri "vless://..."
x-cmd node list
x-cmd node list --subscription <SUBSCRIPTION_NUMBER_OR_NAME>
x-cmd node use <NUMBER_OR_NODE_ID>
x-cmd node delete <NUMBER_OR_NODE_ID>
```

Subscription operations accept the displayed number or exact name. In the interactive subscription menu, Node Management opens the full node menu filtered to that subscription. Node selection and deletion accept the displayed number or an unambiguous ID prefix. Subscription updates replace only nodes owned by that subscription and preserve standalone nodes. Deleting the active node stops a running connection and selects the next available node. Switching nodes while connected restarts the connection with the new node.

## Connection Testing

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <SUBSCRIPTION_NUMBER_OR_NAME> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

This is not a server port check. `x-cmd` starts a temporary xray process for each node and sends an HTTP request through its local SOCKS5 inbound, validating the complete proxy path.

## Running the Proxy

```sh
x-cmd system start
x-cmd system status
x-cmd system stop
x-cmd config set --listen-port 1091

x-cmd proxy enable
x-cmd proxy status
x-cmd proxy disable
```

`system start` uses the active node and exposes an HTTP/SOCKS mixed inbound at `127.0.0.1:1091` by default. Restart after selecting another node. System proxy control updates Windows user Internet Settings, enabled macOS network services, or GNOME `gsettings` on Linux. On Linux it also writes proxy environment variables to the default shell startup file; reopen the shell after enabling or disabling the proxy. It is not a transparent proxy or TUN.

## Updating x-cmd

```sh
x-cmd -v
x-cmd update check
x-cmd update install
```

The updater downloads the current platform artifact from the latest `accloud-proj/x-cmd` Release and replaces the executable. Its directory must be writable. Windows retains the previous executable as `x-cmd.exe.old`.

## GitHub Mirror Management

```sh
x-cmd github-mirror show
x-cmd github-mirror set https://your-mirror.example
x-cmd github-mirror delete
```

`set` selects a custom mirror, while `delete` restores automatic detection. If direct GitHub access then fails, the built-in mirror is selected and persisted again. `x-cmd config show` and `x-cmd config set --github-mirror URL` remain available.

## Uninstall

```sh
x-cmd uninstall
x-cmd uninstall --yes
```

Without `--yes`, the command asks for confirmation and defaults to cancellation. Uninstalling disables the system proxy, stops xray, removes installed shell completion, and deletes `x-cmd` itself, update backups, the configuration file, and runtime data. Stored subscriptions and nodes are permanently deleted as well. When using the interactive menu, select `u. Uninstall` and confirm.

## License

Licensed under the [BSD 3-Clause License](LICENSE).
