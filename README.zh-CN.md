# x-cmd

[![最新版本](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![构建与发布](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![许可证](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go 版本](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

[English](README.md) | 简体中文

`x-cmd` 是一个 xray-core 命令行封装和管理工具。全部操作均支持适合脚本调用的命令参数，无参数运行时则进入交互式菜单。

> **无需直连 GitHub：** 在无法直接访问 GitHub 的地区，安装时可使用 GitHub 镜像前缀，或将 GitHub 主机替换为自建反向代理。安装器会保存该设置，后续下载 xray-core 内核和更新软件时自动继续使用。
>
> 当前可使用替换主机 `github.uzfdafw.cc`，安装时传入 `--github-host github.uzfdafw.cc` 即可。

## 功能

- 安装、查看和切换 xray-core 版本
- 自定义 GitHub 下载地址，适配无法直接访问 GitHub 的地区
- 管理多个 v2rayN 订阅和独立分享链接
- 进行真实代理连接测试，并可自动删除失效节点
- 在 `127.0.0.1:1091` 提供 HTTP/SOCKS mixed 代理
- 启停 xray、查看状态以及控制系统全局代理
- 检查 GitHub Release 并在线更新当前程序
- 构建 Windows、Linux 和 macOS 多架构发布包

## 安装

### Linux 和 macOS

```sh
curl -fsSLO https://raw.githubusercontent.com/accloud-proj/x-cmd/main/scripts/install.sh
sh install.sh
```

指定版本、安装目录或 GitHub 镜像：

```sh
sh install.sh --version v0.2.0 --install-dir "$HOME/.local/bin"
sh install.sh --github-mirror "https://你的GitHub镜像地址"
sh install.sh --github-host "github.uzfdafw.cc"
```

镜像地址会添加到完整 GitHub URL 之前，例如：

```text
https://你的GitHub镜像地址/https://github.com/accloud-proj/x-cmd/releases/latest/download/...
```

`--github-host` 只替换协议和主机，同时保留 GitHub 原始路径，适合使用 Nginx 等反向代理：

```text
https://github.com/accloud-proj/x-cmd/releases/latest/download/...
https://github.uzfdafw.cc/accloud-proj/x-cmd/releases/latest/download/...
```

`--github-mirror` 和 `--github-host` 不能同时使用。未带协议的主机默认使用 HTTPS；只有可信任的本地反代才建议显式传入 `http://host`。

安装脚本本身也可以通过替换后的主机获取：

```sh
curl -fsSL https://github.uzfdafw.cc/accloud-proj/x-cmd/releases/latest/download/install.sh -o install.sh
sh install.sh --github-host github.uzfdafw.cc
```

反向代理必须保留这些路径，并改写或代理 GitHub Release 资产重定向；否则客户端可能再次跳转到 GitHub 的资产域名。

如果 GitHub 本身无法访问，可以先通过镜像中的 Release 地址下载安装脚本：

```sh
MIRROR="https://你的GitHub镜像地址"
curl -fsSL "$MIRROR/https://github.com/accloud-proj/x-cmd/releases/latest/download/install.sh" -o install.sh
sh install.sh --github-mirror "$MIRROR"
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/accloud-proj/x-cmd/main/scripts/install.ps1 -OutFile install.ps1
.\install.ps1
```

可选参数：

```powershell
.\install.ps1 -Version v0.2.0 -InstallDir "$env:LOCALAPPDATA\x-cmd\bin"
.\install.ps1 -GitHubMirror "https://你的GitHub镜像地址"
.\install.ps1 -GitHubHost "github.uzfdafw.cc"
```

同一主机可通过 `https://github.uzfdafw.cc/accloud-proj/x-cmd/releases/latest/download/install.ps1` 提供安装脚本。

通过镜像获取 PowerShell 安装脚本：

```powershell
$mirror = "https://你的GitHub镜像地址"
Invoke-WebRequest "$mirror/https://github.com/accloud-proj/x-cmd/releases/latest/download/install.ps1" -OutFile install.ps1
.\install.ps1 -GitHubMirror $mirror
```

两个安装脚本都会使用同一 Release 的 `checksums.txt`，并在安装前校验压缩包 SHA-256。

### 从源码构建

需要 Go 1.27 或更高版本：

```sh
go build -o x-cmd .
```

无参数运行 `x-cmd` 可进入交互菜单。源码版本保存在 `version.Version` 变量中，标签构建会通过 `-ldflags` 注入标签版本。

## 兼容性

### 节点协议

| 协议           | 接受的链接                           | 认证信息                | 传输与安全层                                       |
| -------------- | ------------------------------------ | ----------------------- | -------------------------------------------------- |
| VMess          | `vmess://` Base64 编码的 v2rayN JSON | UUID、alterId、security | TCP、WebSocket、gRPC、HTTP/2；none 或 TLS          |
| VLESS          | `vless://` URI                       | UUID、encryption、flow  | TCP、WebSocket、gRPC、HTTP/2；none、TLS 或 REALITY |
| Trojan         | `trojan://` URI                      | 密码、可选 flow         | TCP、WebSocket、gRPC、HTTP/2；none、TLS 或 REALITY |
| Shadowsocks    | `ss://` SIP002 URI                   | 加密方法和密码          | xray 支持的 AEAD 方法；不支持插件                  |
| HTTP/HTTPS     | `http://` 或 `https://` URI          | 可选用户名/密码         | HTTP 上游；HTTPS 对上游启用 TLS                    |
| SOCKS5         | `socks://` 或 `socks5://` URI        | 可选用户名/密码         | Xray SOCKS5 出站                                   |
| 任意 Xray 出站 | `xray://` Base64URL JSON             | 由 JSON 定义            | 原生出站对象，无损传递给 xray                      |

在适用协议中，WebSocket host/path、gRPC service name、HTTP/2 host/path、TLS SNI，以及 REALITY 的 `fp`、`pbk`、`sid`、`spx` 参数会被转换为 xray 配置。

原生 `xray://` 格式覆盖 xray-core 的全部出站协议，包括 WireGuard、Hysteria、Freedom、DNS、Blackhole、Loopback，并能兼容以后新增的协议。将一个完整 Xray `OutboundObject` 编码为无填充 Base64URL，可选追加 `#名称`：

```text
xray://eyJwcm90b2NvbCI6IndpcmVndWFyZCIsInNldHRpbmdzIjp7Li4ufX0#我的WireGuard
```

其中必须包含 `protocol` 和 `settings`。对象会原样传给 xray，因此高级用户可以包含 `streamSettings`、`mux` 等其他出站字段。Blackhole、DNS、Freedom、Loopback 是 Xray 路由出站而非远程代理服务器；格式上支持，但通常不适合作为订阅节点选择。

WireGuard 和 Xray Hysteria 的完整配置没有一种可覆盖所有字段的统一 v2rayN 分享 URI，因此应使用 `xray://`，而不是猜测或丢失字段的链接映射。TUIC 不是 xray-core 出站协议。仍不接受 Shadowsocks SIP008 JSON/插件和完整 Xray 配置文件；`xray://` 只包含一个出站对象。

### 订阅格式

仅接受 v2rayN 订阅格式：

1. 整份内容经过 Base64 编码，每行一个分享 URI。
2. 未编码纯文本，分享 URI 之间使用空白字符分隔。
3. URI 必须是上表中的 VMess、VLESS、Trojan、Shadowsocks、HTTP(S)、SOCKS5 或原生 `xray://` 格式。

不会解析 Clash YAML、Clash.Meta/Mihomo YAML、sing-box JSON、Surge 配置、SIP008 JSON 或嵌套订阅。无效行会被跳过；如果没有剩余的受支持节点，则更新失败。

## 内核管理

```sh
x-cmd core show
x-cmd core install --version v25.8.3
x-cmd core install --version v25.8.3 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
x-cmd config set --download-url "https://你的镜像地址/https://github.com/XTLS/Xray-core/releases/download"
```

## 订阅与节点管理

```sh
x-cmd sub add --name "订阅 A" --url "https://example.com/subscription"
x-cmd sub list
x-cmd sub edit <ID> --name "新名称" --url "https://example.com/new"
x-cmd sub update <ID>
x-cmd sub update all
x-cmd sub nodes <ID>
x-cmd sub delete <ID>

x-cmd node add --uri "vless://..."
x-cmd node list
x-cmd node list --subscription <订阅ID>
x-cmd node use <节点ID>
x-cmd node delete <节点ID>
```

可以使用能够唯一匹配的 ID 前缀。更新订阅只替换该订阅所属节点，并保留独立导入节点。第一个可用节点会被自动选中。

## 连接测试

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <ID> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

这不是服务器端口探测。`x-cmd` 会为每个节点启动临时 xray 进程，并通过其本地 SOCKS5 入站发送 HTTP 请求，以验证完整代理链路。

## 运行代理

```sh
x-cmd start
x-cmd status
x-cmd stop
x-cmd config set --listen-port 1091

x-cmd proxy enable
x-cmd proxy status
x-cmd proxy disable
```

`start` 使用活动节点，默认在 `127.0.0.1:1091` 提供 HTTP/SOCKS mixed 入站。选择其他节点后需要重启连接。系统代理功能会修改 Windows 当前用户 Internet Settings、macOS 已启用网络服务，或 Linux GNOME `gsettings`。该功能不是透明代理或 TUN。

## 更新 x-cmd

```sh
x-cmd update check
x-cmd update install
```

更新器从 `accloud-proj/x-cmd` 最新 Release 下载当前系统和架构对应的资产，并替换当前程序。安装目录必须可写。Windows 会将旧程序保留为 `x-cmd.exe.old`。

## 自动构建与发布

[发布工作流](.github/workflows/release.yml)构建 Windows amd64/arm64/386、Linux amd64/arm64/armv7 和 macOS amd64/arm64。推送例如 `v0.2.0` 的标签会创建 Release，并上传各平台压缩包及 `checksums.txt`。

## 数据目录

数据保存在操作系统用户配置目录下的 `x-cmd/config.json`。可通过 `X_CMD_CONFIG` 覆盖路径。订阅链接可能包含凭据，请勿公开该文件。

## 许可证

本项目采用 [BSD 3-Clause License](LICENSE)。
