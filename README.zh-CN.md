# x-cmd

[![最新版本](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![构建与发布](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![许可证](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go 版本](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

[English](README.md) | 简体中文

`x-cmd` 是一个 xray-core 命令行封装和管理工具。全部操作均支持适合脚本调用的命令参数，无参数运行时则进入交互式菜单。

> **无需直连 GitHub：** 安装器和客户端内置镜像加速能力。未手动指定镜像时会优先直连 GitHub，连接失败后自动切换内置镜像，也支持通过参数手动切换镜像。

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
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh)
```

指定版本或 GitHub 镜像：

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh) --version v0.7.0
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh) --github-mirror https://your-mirror.example
```

如果当前网络无法访问 GitHub，也可以在其他可访问仓库内容的环境中打开 [scripts/install.sh](scripts/install.sh)，将文件内容完整粘贴到本机的 `install.sh` 后执行：

```sh
bash install.sh
```

### Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.ps1 -OutFile install.ps1
.\install.ps1
```

可选参数：

```powershell
.\install.ps1 -Version v0.7.0
.\install.ps1 -GitHubMirror https://your-mirror.example
```

如果当前网络无法访问 GitHub，也可以在其他可访问仓库内容的环境中打开 [scripts/install.ps1](scripts/install.ps1)，将文件内容完整粘贴到本机的 `install.ps1` 后执行：

```powershell
.\install.ps1
```

## 兼容性

| 协议           | 接受的链接                           | 认证信息                | 传输与安全层                                       |
| -------------- | ------------------------------------ | ----------------------- | -------------------------------------------------- |
| VMess          | `vmess://` Base64 编码的 v2rayN JSON | UUID、alterId、security | TCP、WebSocket、gRPC、HTTP/2；none 或 TLS          |
| VLESS          | `vless://` URI                       | UUID、encryption、flow  | TCP、WebSocket、gRPC、HTTP/2；none、TLS 或 REALITY |
| Trojan         | `trojan://` URI                      | 密码、可选 flow         | TCP、WebSocket、gRPC、HTTP/2；none、TLS 或 REALITY |
| Shadowsocks    | `ss://` SIP002 URI                   | 加密方法和密码          | xray 支持的 AEAD 方法；不支持插件                  |
| HTTP/HTTPS     | `http://` 或 `https://` URI          | 可选用户名/密码         | HTTP 上游；HTTPS 对上游启用 TLS                    |
| SOCKS5         | `socks://` 或 `socks5://` URI        | 可选用户名/密码         | Xray SOCKS5 出站                                   |
| 任意 Xray 出站 | `xray://` Base64URL JSON             | 由 JSON 定义            | 原生出站对象，无损传递给 xray                      |

支持 Base64 编码或纯文本的 v2rayN 订阅，节点协议以表格所列类型为准。

## 命令行自动补全

```sh
x-cmd completion install
x-cmd completion uninstall
```

默认自动识别当前 Shell，也可以明确指定 Bash、Zsh、Fish 或 PowerShell，例如 `x-cmd completion install powershell`。安装后重新打开终端即可生效。

## 内核管理

```sh
x-cmd core show
x-cmd core releases
x-cmd core install --version v26.3.27
x-cmd core install --version v26.3.27 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
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
x-cmd node use <序号或节点ID>
x-cmd node delete <序号或节点ID>
```

选择和删除节点时可使用列表序号或能够唯一匹配的 ID 前缀。更新订阅只替换该订阅所属节点，并保留独立导入节点。删除活动节点会停止正在运行的连接并自动选择下一个可用节点；连接运行时切换节点会使用新节点自动重启连接。

## 连接测试

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <ID> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

这不是服务器端口探测。`x-cmd` 会为每个节点启动临时 xray 进程，并通过其本地 SOCKS5 入站发送 HTTP 请求，以验证完整代理链路。

## 运行代理

```sh
x-cmd system start
x-cmd system status
x-cmd system stop
x-cmd config set --listen-port 1091

x-cmd proxy enable
x-cmd proxy status
x-cmd proxy disable
```

`system start` 使用活动节点，默认在 `127.0.0.1:1091` 提供 HTTP/SOCKS mixed 入站。选择其他节点后需要重启连接。系统代理功能会修改 Windows 当前用户 Internet Settings、macOS 已启用网络服务，或 Linux GNOME `gsettings`。该功能不是透明代理或 TUN。

## 更新 x-cmd

```sh
x-cmd -v
x-cmd update check
x-cmd update install
```

更新器从 `accloud-proj/x-cmd` 最新 Release 下载当前系统和架构对应的资产，并替换当前程序。安装目录必须可写。Windows 会将旧程序保留为 `x-cmd.exe.old`。

## GitHub 镜像管理

```sh
x-cmd github-mirror show
x-cmd github-mirror set https://your-mirror.example
x-cmd github-mirror delete
```

`set` 会切换到指定镜像，`delete` 会恢复自动模式（优先直连 GitHub，失败后使用内置镜像）。也可以继续使用 `x-cmd config show` 和 `x-cmd config set --github-mirror URL`。

## 卸载

```sh
x-cmd uninstall
x-cmd uninstall --yes
```

不带 `--yes` 时会询问是否确认，默认不删除。卸载时会关闭系统代理、停止 xray、自动卸载命令行补全，并删除 `x-cmd` 自身、更新备份、配置文件和运行数据。配置中的订阅与节点也会一并永久删除。无参数运行交互菜单时，也可以选择 `u. 卸载` 并确认。

## 许可证

本项目采用 [BSD 3-Clause License](LICENSE)。
