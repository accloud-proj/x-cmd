# x-cmd

[![最新版本](https://img.shields.io/github/v/release/accloud-proj/x-cmd?display_name=tag&sort=semver)](https://github.com/accloud-proj/x-cmd/releases/latest)
[![构建与发布](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml/badge.svg)](https://github.com/accloud-proj/x-cmd/actions/workflows/release.yml)
[![许可证](https://img.shields.io/github/license/accloud-proj/x-cmd)](LICENSE)
[![Go 版本](https://img.shields.io/github/go-mod/go-version/accloud-proj/x-cmd)](go.mod)

[English](README.md) | 简体中文

`x-cmd` 是一个 xray-core 命令行封装和管理工具。全部操作均支持适合脚本调用的命令参数，无参数运行时则进入交互式菜单。

> **无需直连 GitHub：** 未配置镜像时，如果 GitHub 直连失败或低于 100 kbit/s，客户端会自动使用内置镜像。只要已有镜像配置，就会保持原值并跳过检测。

## 功能

- 安装、查看和切换 xray-core 版本
- 自定义 GitHub 下载地址，适配无法直接访问 GitHub 的地区
- 管理多个 v2rayN 订阅和独立分享链接
- 进行真实代理连接测试，并可自动删除失效节点
- 在 `127.0.0.1:1091` 提供 HTTP/SOCKS mixed 代理
- 启停 xray、查看状态，并进入已配置代理的子 Shell
- 检查 GitHub Release 并在线更新当前程序
- 构建 Windows、Linux、macOS、FreeBSD 和 OpenBSD 多架构发布包

## 安装

### Linux 和 macOS

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/accloud-proj/x-cmd/master/scripts/install.sh)
```

安装脚本默认下载最新 Release。使用 GitHub 镜像：

```sh
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
.\install.ps1 -GitHubMirror https://your-mirror.example
```

如果当前网络无法访问 GitHub，也可以在其他可访问仓库内容的环境中打开 [scripts/install.ps1](scripts/install.ps1)，将文件内容完整粘贴到本机的 `install.ps1` 后执行：

```powershell
.\install.ps1
```

## 支持的链接

支持 VMess、VLESS、Trojan、Shadowsocks、HTTP/HTTPS、SOCKS5 和原生 `xray://` 出站链接。订阅支持 Base64 编码或纯文本 v2rayN 格式。

## 命令行自动补全

```sh
x-cmd completion install
x-cmd completion uninstall
```

默认自动识别当前 Shell，也可以明确指定 Bash、Zsh、Fish 或 PowerShell，例如 `x-cmd completion install powershell`。安装后重新打开终端即可生效。

## GitHub 镜像管理

```sh
x-cmd github-mirror show
x-cmd github-mirror set https://your-mirror.example
x-cmd github-mirror delete
```

`set` 设置镜像后不会再测速或自动修改。`delete` 恢复自动模式，GitHub 无法访问或低于 100 kbit/s 时会使用内置镜像。

## 内核管理

```sh
x-cmd core show
x-cmd core releases
x-cmd core install --version v26.3.27
x-cmd core install --version v26.3.27 --dir /path/to/xray
x-cmd config set --xray-path /path/to/xray
```

`core releases` 仅显示 xray-core 稳定版本，草稿和预览版本会被排除。

## 订阅与节点管理

```sh
x-cmd sub add --name "订阅 A" --url "https://example.com/subscription"
x-cmd sub list
x-cmd sub edit <序号或名称> --name "新名称" --url "https://example.com/new"
x-cmd sub update <序号或名称>
x-cmd sub update all
x-cmd sub nodes <序号或名称>
x-cmd sub delete <序号或名称>

x-cmd node add --uri "vless://..."
x-cmd node list
x-cmd node list --subscription <订阅序号或名称>
x-cmd node use <序号或节点ID>
x-cmd node delete <序号或节点ID>
```

订阅可使用列表序号或名称，节点可使用序号或 ID。连接运行时切换活动节点会自动重启连接。

## 连接测试

```sh
x-cmd node test --timeout 10s
x-cmd node test --subscription <订阅序号或名称> --timeout 10s --delete-invalid
x-cmd config set --test-url "https://example.com/generate_204"
```

使用 `--delete-invalid` 可自动删除测试失败的节点。

## 运行代理

```sh
x-cmd node list
x-cmd node use <节点序号或ID>
x-cmd system start
x-cmd system status
x-cmd system stop
x-cmd config set --listen-port 1091
x-cmd config set --allow-lan=true

x-cmd shell
x-cmd shell fish
eval "$(x-cmd activate bash)"
```

Fish 或 PowerShell 7 可按以下方式激活当前 Shell：

```fish
x-cmd activate fish | source
```

```powershell
Invoke-Expression (& x-cmd activate pwsh | Out-String)
```

默认监听 1091 mixed 端口，被占用时会自动改用可用端口。支持开启局域网监听（默认关闭），环境支持时会同时启用 IPv4 和 IPv6。使用 `shell` 或 `activate` 前必须先启动连接。

## 更新 x-cmd

```sh
x-cmd -v
x-cmd update check
x-cmd update install
```

`update check` 检查新版本，`update install` 安装新版本。

## 卸载

```sh
x-cmd uninstall
x-cmd uninstall --yes
```

不带 `--yes` 时会请求确认。卸载会删除程序、命令补全、配置、订阅和节点。

## 本地构建

使用 PowerShell 7 构建 Windows amd64：

```powershell
pwsh ./scripts/build.ps1
```

使用 Shell 构建 Linux amd64：

```sh
sh ./scripts/build.sh
```

构建结果分别位于 `dist/windows-amd64` 和 `dist/linux-amd64`。两个脚本都会先运行测试。

## 许可证

本项目采用 [BSD 3-Clause License](LICENSE)。
