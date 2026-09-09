# x-cmd

[![Latest Release](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![Build and Release](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

English | [简体中文](README.zh-CN.md)

`x-cmd` is a command-line wrapper and manager for xray-core. Every operation is available as a script-friendly command, while running it without arguments opens an interactive menu.

> **GitHub access is optional:** When no mirror is configured, the client switches to the built-in mirror if direct access fails or is slower than 100 kbit/s. An existing mirror setting is always kept and skips this check.

## Features

- Install, inspect, and switch xray-core versions
- Override GitHub download URLs for regions where GitHub is unavailable
- Manage multiple v2rayN subscriptions and standalone share links
- Perform real proxy connection tests and optionally remove failed nodes
- Run an HTTP/SOCKS mixed proxy on `127.0.0.1:1091`
- Start and stop xray, then open a proxy-configured child shell
- Check GitHub Releases and update the current executable online
- Build Windows, Linux, macOS, FreeBSD, and OpenBSD release artifacts on multiple architectures

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

## Supported Links

VMess, VLESS, Trojan, Shadowsocks, HTTP/HTTPS, SOCKS5, and native `xray://` outbound links are supported. Subscriptions can use Base64-encoded or plain-text v2rayN format.

## Shell Completion

```sh
x-cmd completion install
x-cmd completion uninstall
```

The current shell is detected automatically. Bash, Zsh, Fish, and PowerShell can also be selected explicitly, for example `x-cmd completion install powershell`. Reopen the terminal after installation.

## GitHub Mirror Management

```sh
x-cmd github-mirror show
x-cmd github-mirror set https://your-mirror.example
x-cmd github-mirror delete
```

`set` keeps the chosen mirror without further speed checks. `delete` returns to automatic mode, which uses the built-in mirror when GitHub is unreachable or slower than 100 kbit/s.

## Core Management

```sh
x-cmd core show
x-cmd core releases
x-cmd core install --version v26.3.27
x-cmd core install --version v26.3.27 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
```

Only stable xray-core releases are shown by `core releases`; draft and prerelease versions are excluded.

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

Subscriptions accept their displayed number or name. Nodes accept their number or ID. Switching the active node while connected restarts the connection automatically.

## Connection Testing

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <SUBSCRIPTION_NUMBER_OR_NAME> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

Use `--delete-invalid` to remove nodes that fail the connection test.

## Running the Proxy

```sh
x-cmd node list
x-cmd node use <NUMBER_OR_NODE_ID>
x-cmd system start
x-cmd system status
x-cmd system stop
x-cmd config set --listen-port 1091
x-cmd config set --allow-lan=true

x-cmd shell
x-cmd shell fish
eval "$(x-cmd activate bash)"
```

For Fish or PowerShell 7, activate the current shell with:

```fish
x-cmd activate fish | source
```

```powershell
Invoke-Expression (& x-cmd activate pwsh | Out-String)
```

The mixed proxy uses port 1091 by default and automatically switches to an available port when necessary. LAN access is optional and disabled by default; IPv4 and IPv6 are both enabled when supported. Start the connection before using `shell` or `activate`.

## Updating x-cmd

```sh
x-cmd -v
x-cmd update check
x-cmd update install
```

`update check` checks for a newer version; `update install` installs it.

## Uninstall

```sh
x-cmd uninstall
x-cmd uninstall --yes
```

Without `--yes`, the command asks for confirmation. Uninstall removes the program, completion, configuration, subscriptions, and nodes.

## Local Build

Build Windows amd64 with PowerShell 7:

```powershell
pwsh ./scripts/build.ps1
```

Build Linux amd64 with Shell:

```sh
sh ./scripts/build.sh
```

Outputs are written to `dist/windows-amd64` and `dist/linux-amd64`. Both scripts run tests before building.

## License

Licensed under the [BSD 3-Clause License](LICENSE).
