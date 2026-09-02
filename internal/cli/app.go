package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/accloud-proj/x-cmd/internal/completion"
	"github.com/accloud-proj/x-cmd/internal/githuburl"
	"github.com/accloud-proj/x-cmd/internal/nodes"
	"github.com/accloud-proj/x-cmd/internal/state"
	"github.com/accloud-proj/x-cmd/internal/systemproxy"
	"github.com/accloud-proj/x-cmd/internal/uninstaller"
	"github.com/accloud-proj/x-cmd/internal/updater"
	"github.com/accloud-proj/x-cmd/internal/version"
	"github.com/accloud-proj/x-cmd/internal/xray"
)

type App struct {
	store     *state.Store
	input     *bufio.Reader
	output    io.Writer
	uninstall func(string, string) error
	pause     func(time.Duration)
}

func New() *App {
	path, _ := state.DefaultPath()
	return &App{store: state.New(path), input: bufio.NewReader(os.Stdin), output: os.Stdout, pause: time.Sleep}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.interactive()
	}
	switch args[0] {
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	case "version", "-v", "--version":
		fmt.Fprintf(a.output, "x-cmd %s\n", version.Version)
		return nil
	case "system":
		return a.system(args[1:])
	case "proxy":
		return a.proxy(args[1:])
	case "update":
		return a.update(args[1:])
	case "uninstall":
		return a.uninstallApp(args[1:])
	case "completion":
		return a.completion(args[1:])
	case "core":
		return a.core(args[1:])
	case "config":
		return a.config(args[1:])
	case "github-mirror":
		return a.githubMirror(args[1:])
	case "sub", "subscription":
		return a.subscriptions(args[1:])
	case "node":
		return a.node(args[1:])
	default:
		return fmt.Errorf("未知命令 %q，运行 x-cmd help 查看帮助", args[0])
	}
}

func (a *App) system(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("用法: x-cmd system start|status|stop")
	}
	switch args[0] {
	case "start", "status", "stop":
		return a.service(args[0])
	default:
		return fmt.Errorf("用法: x-cmd system start|status|stop")
	}
}

func (a *App) uninstallApp(args []string) error {
	flags := newFlagSet("uninstall")
	confirmed := flags.Bool("yes", false, "确认删除程序和全部配置")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("用法: x-cmd uninstall [--yes]")
	}
	if !*confirmed {
		answer := a.prompt("卸载会删除程序和全部配置，确认? [y/N]")
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			fmt.Fprintln(a.output, "[提示] 已取消卸载")
			return nil
		}
	}
	if err := a.service("stop"); err != nil {
		return err
	}
	if err := completion.UninstallAll(); err != nil {
		return fmt.Errorf("卸载命令补全失败: %w", err)
	}
	remove := a.uninstall
	if remove == nil {
		remove = uninstaller.Remove
	}
	if err := remove(a.store.Path(), a.store.RuntimeDir()); err != nil {
		return err
	}
	fmt.Fprintln(a.output, "[成功] 卸载完成")
	return nil
}

func (a *App) completion(args []string) error {
	if len(args) > 0 && args[0] == "candidates" {
		for _, candidate := range completion.Candidates(args[1:]) {
			fmt.Fprintln(a.output, candidate)
		}
		return nil
	}
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("用法: x-cmd completion install|uninstall [bash|zsh|fish|powershell]")
	}
	shell := ""
	if len(args) == 2 {
		shell = args[1]
	}
	switch args[0] {
	case "install":
		paths, err := completion.Install(shell)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 命令补全安装完成")
		for _, path := range paths {
			fmt.Fprintln(a.output, "-", path)
		}
		fmt.Fprintln(a.output, "[提示] 请重新打开终端使补全生效")
		return nil
	case "uninstall":
		if err := completion.Uninstall(shell); err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 命令补全已卸载")
		return nil
	default:
		return fmt.Errorf("用法: x-cmd completion install|uninstall [bash|zsh|fish|powershell]")
	}
}

func (a *App) core(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(a.output, "配置版本: %s\n内核路径: %s\n下载地址: %s\n", emptyAs(data.Settings.XrayVersion, "未配置"), emptyAs(data.Settings.XrayPath, "PATH 中的 xray"), data.Settings.DownloadURL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		version, err := xray.Version(ctx, data.Settings.XrayPath)
		if err != nil {
			fmt.Fprintln(a.output, "实际版本: 不可用 ("+err.Error()+")")
			return nil
		}
		fmt.Fprintln(a.output, "实际版本:", version)
		return nil
	}
	if args[0] == "releases" {
		return a.xrayReleases()
	}
	if args[0] != "install" {
		return fmt.Errorf("用法: x-cmd core [show|releases|install]")
	}
	flags := newFlagSet("core install")
	version := flags.String("version", "", "Xray Release 版本")
	directory := flags.String("dir", "", "安装目录")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	rewriters, err := githuburl.Candidates(data.Settings.GitHubMirror)
	if err != nil {
		return err
	}
	var binary string
	for index, rewriter := range rewriters {
		downloadURL, rewriteErr := rewriter.Rewrite(data.Settings.DownloadURL)
		if rewriteErr != nil {
			return rewriteErr
		}
		if index > 0 {
			fmt.Fprintln(a.output, "[提示] GitHub 直连不可用，正在切换到内置镜像...")
		}
		fmt.Fprintf(a.output, "[信息] 正在下载 Xray %s...\n", *version)
		binary, err = xray.Install(context.Background(), *version, downloadURL, *directory)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	data.Settings.XrayVersion = *version
	data.Settings.XrayPath = binary
	if err := a.store.Save(data); err != nil {
		return err
	}
	fmt.Fprintln(a.output, "[成功] Xray 内核安装完成:", binary)
	return nil
}

func (a *App) xrayReleases() error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	rewriters, err := githuburl.Candidates(data.Settings.GitHubMirror)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var releases []xray.Release
	for index, rewriter := range rewriters {
		endpoint, rewriteErr := rewriter.Rewrite("https://api.github.com/repos/XTLS/Xray-core/releases?per_page=10")
		if rewriteErr != nil {
			return rewriteErr
		}
		if index > 0 {
			fmt.Fprintln(a.output, "[提示] GitHub 直连不可用，正在切换到内置镜像...")
		}
		releases, err = xray.RecentReleases(ctx, endpoint, 5)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		return fmt.Errorf("未获取到可用的 Xray Release")
	}
	fmt.Fprintln(a.output, "\n[信息] 最近的 Xray Release:")
	for index, release := range releases {
		fmt.Fprintf(a.output, "%d. %-16s %s\n", index+1, release.TagName, release.PublishedAt.Local().Format("2006-01-02"))
	}
	return nil
}

func (a *App) config(args []string) error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "show" {
		fmt.Fprintf(a.output, "download-url: %s\ngithub-mirror: %s\nxray-path: %s\ntest-url: %s\nlisten-port: %d\n", data.Settings.DownloadURL, emptyAs(data.Settings.GitHubMirror, "自动（直连失败时使用内置镜像）"), data.Settings.XrayPath, data.Settings.TestURL, data.Settings.ListenPort)
		return nil
	}
	if args[0] != "set" {
		return fmt.Errorf("用法: x-cmd config [show|set]")
	}
	flags := newFlagSet("config set")
	downloadURL := flags.String("download-url", "", "xray release 下载基地址")
	githubMirror := flags.String("github-mirror", "", "GitHub 镜像前缀")
	xrayPath := flags.String("xray-path", "", "xray 可执行文件路径")
	testURL := flags.String("test-url", "", "节点测试 URL")
	listenPort := flags.Int("listen-port", 1091, "本地 mixed 代理端口")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	changed := map[string]bool{}
	flags.Visit(func(item *flag.Flag) { changed[item.Name] = true })
	if changed["download-url"] {
		data.Settings.DownloadURL = strings.TrimRight(*downloadURL, "/")
	}
	if changed["github-mirror"] {
		rewriter, err := githuburl.New(*githubMirror)
		if err != nil {
			return err
		}
		data.Settings.GitHubMirror = rewriter.Mirror
	}
	if changed["xray-path"] {
		data.Settings.XrayPath = *xrayPath
	}
	if changed["test-url"] {
		data.Settings.TestURL = *testURL
	}
	if changed["listen-port"] {
		if *listenPort < 1 || *listenPort > 65535 {
			return fmt.Errorf("监听端口必须在 1 到 65535 之间")
		}
		data.Settings.ListenPort = *listenPort
	}
	if len(changed) == 0 {
		return fmt.Errorf("至少指定一个要修改的配置项")
	}
	return a.store.Save(data)
}

func (a *App) githubMirror(args []string) error {
	action := "show"
	if len(args) > 0 {
		action = args[0]
	}
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	switch action {
	case "show":
		fmt.Fprintln(a.output, emptyAs(data.Settings.GitHubMirror, "自动（直连失败时使用内置镜像）"))
		return nil
	case "set":
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("用法: x-cmd github-mirror set <URL>")
		}
		rewriter, err := githuburl.New(args[1])
		if err != nil {
			return err
		}
		data.Settings.GitHubMirror = rewriter.Mirror
	case "delete", "rm":
		if len(args) != 1 {
			return fmt.Errorf("用法: x-cmd github-mirror delete")
		}
		data.Settings.GitHubMirror = ""
	default:
		return fmt.Errorf("用法: x-cmd github-mirror show|set <URL>|delete")
	}
	return a.store.Save(data)
}

func (a *App) subscriptions(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		return a.listSubscriptions()
	}
	switch args[0] {
	case "add":
		flags := newFlagSet("sub add")
		name := flags.String("name", "", "订阅名称")
		address := flags.String("url", "", "订阅链接")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *name == "" || *address == "" {
			return fmt.Errorf("--name 和 --url 均为必填")
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		subscription := state.Subscription{ID: state.NewID(), Name: *name, URL: *address}
		data.Subscriptions = append(data.Subscriptions, subscription)
		if err := a.store.Save(data); err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 已添加订阅:", subscription.ID)
		return a.updateSubscriptions(subscription.ID)
	case "edit":
		if len(args) < 2 {
			return fmt.Errorf("用法: x-cmd sub edit <ID> [--name 名称] [--url 链接]")
		}
		flags := newFlagSet("sub edit")
		name := flags.String("name", "", "订阅名称")
		address := flags.String("url", "", "订阅链接")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		changed := map[string]bool{}
		flags.Visit(func(item *flag.Flag) { changed[item.Name] = true })
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		index, err := findSubscription(data, args[1])
		if err != nil {
			return err
		}
		if changed["name"] {
			data.Subscriptions[index].Name = *name
		}
		if changed["url"] {
			data.Subscriptions[index].URL = *address
		}
		if len(changed) == 0 {
			return fmt.Errorf("至少指定 --name 或 --url")
		}
		return a.store.Save(data)
	case "delete", "rm":
		if len(args) != 2 {
			return fmt.Errorf("用法: x-cmd sub delete <ID>")
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		index, err := findSubscription(data, args[1])
		if err != nil {
			return err
		}
		id := data.Subscriptions[index].ID
		for _, node := range data.Nodes {
			if node.SubscriptionID == id && node.ID == data.Settings.ActiveNodeID {
				return fmt.Errorf("该订阅包含活动节点，请先使用 node use 切换节点")
			}
		}
		data.Subscriptions = append(data.Subscriptions[:index], data.Subscriptions[index+1:]...)
		data.Nodes = filterNodes(data.Nodes, func(node state.Node) bool { return node.SubscriptionID != id })
		return a.store.Save(data)
	case "update":
		id := ""
		if len(args) > 1 {
			id = args[1]
		}
		return a.updateSubscriptions(id)
	case "nodes":
		if len(args) != 2 {
			return fmt.Errorf("用法: x-cmd sub nodes <ID>")
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		index, err := findSubscription(data, args[1])
		if err != nil {
			return err
		}
		return a.printNodes(data, data.Subscriptions[index].ID)
	default:
		return fmt.Errorf("未知订阅命令 %q", args[0])
	}
}

func (a *App) listSubscriptions() error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	writer := tabwriter.NewWriter(a.output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\t名称\t节点数\t更新时间\t链接")
	for _, subscription := range data.Subscriptions {
		count := 0
		for _, node := range data.Nodes {
			if node.SubscriptionID == subscription.ID {
				count++
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%d\t%s\t%s\n", subscription.ID, subscription.Name, count, formatTime(subscription.UpdatedAt), subscription.URL)
	}
	return writer.Flush()
}

func (a *App) updateSubscriptions(id string) error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	var targets []int
	if id == "" || id == "all" {
		for index := range data.Subscriptions {
			targets = append(targets, index)
		}
	} else {
		index, err := findSubscription(data, id)
		if err != nil {
			return err
		}
		targets = append(targets, index)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, index := range targets {
		subscription := &data.Subscriptions[index]
		request, err := http.NewRequest(http.MethodGet, subscription.URL, nil)
		if err != nil {
			return err
		}
		request.Header.Set("User-Agent", "x-cmd/"+version.Version)
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("更新订阅 %s 失败: %w", subscription.Name, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 16<<20))
		response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("更新订阅 %s 失败: HTTP %s", subscription.Name, response.Status)
		}
		links := nodes.DecodeV2RayNSubscription(body)
		if len(links) == 0 {
			return fmt.Errorf("订阅 %s 中没有可识别节点", subscription.Name)
		}
		oldIDs := map[string]string{}
		for _, node := range data.Nodes {
			if node.SubscriptionID == subscription.ID {
				oldIDs[node.URI] = node.ID
			}
		}
		data.Nodes = filterNodes(data.Nodes, func(node state.Node) bool { return node.SubscriptionID != subscription.ID })
		now := time.Now()
		for _, link := range links {
			parsed, _ := nodes.Parse(link)
			nodeID := oldIDs[link]
			if nodeID == "" {
				nodeID = state.NewID()
			}
			data.Nodes = append(data.Nodes, state.Node{ID: nodeID, Name: parsed.Name, URI: link, SubscriptionID: subscription.ID, UpdatedAt: now})
		}
		subscription.UpdatedAt = now
		fmt.Fprintf(a.output, "[成功] 已更新订阅 %s，共 %d 个节点\n", subscription.Name, len(links))
	}
	normalizeActiveNode(&data)
	return a.store.Save(data)
}

func (a *App) node(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		flags := newFlagSet("node list")
		subscription := flags.String("subscription", "", "订阅 ID")
		if len(args) > 0 {
			if err := flags.Parse(args[1:]); err != nil {
				return err
			}
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		if *subscription != "" {
			index, err := findSubscription(data, *subscription)
			if err != nil {
				return err
			}
			*subscription = data.Subscriptions[index].ID
		}
		return a.printNodes(data, *subscription)
	}
	switch args[0] {
	case "add":
		flags := newFlagSet("node add")
		name := flags.String("name", "", "节点名称")
		uri := flags.String("uri", "", "分享链接")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *uri == "" && flags.NArg() > 0 {
			*uri = flags.Arg(0)
		}
		parsed, err := nodes.Parse(*uri)
		if err != nil {
			return err
		}
		if *name == "" {
			*name = parsed.Name
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		node := state.Node{ID: state.NewID(), Name: *name, URI: *uri, UpdatedAt: time.Now()}
		data.Nodes = append(data.Nodes, node)
		if data.Settings.ActiveNodeID == "" {
			data.Settings.ActiveNodeID = node.ID
		}
		if err := a.store.Save(data); err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 已添加节点:", node.ID)
		return nil
	case "delete", "rm":
		if len(args) != 2 {
			return fmt.Errorf("用法: x-cmd node delete <ID>")
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		index, err := findNode(data, args[1])
		if err != nil {
			return err
		}
		if data.Nodes[index].ID == data.Settings.ActiveNodeID {
			return fmt.Errorf("不能删除活动节点，请先使用 node use <ID> 切换")
		}
		data.Nodes = append(data.Nodes[:index], data.Nodes[index+1:]...)
		return a.store.Save(data)
	case "use", "select":
		if len(args) != 2 {
			return fmt.Errorf("用法: x-cmd node use <序号或 ID>")
		}
		data, err := a.store.Load()
		if err != nil {
			return err
		}
		index, err := findNodeSelection(data, args[1])
		if err != nil {
			return err
		}
		data.Settings.ActiveNodeID = data.Nodes[index].ID
		if err := a.store.Save(data); err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 当前活动节点:", data.Nodes[index].Name)
		return nil
	case "test":
		flags := newFlagSet("node test")
		removeInvalid := flags.Bool("delete-invalid", false, "自动删除失败节点")
		subscription := flags.String("subscription", "", "仅测试指定订阅")
		timeout := flags.Duration("timeout", 10*time.Second, "单节点超时")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		return a.testNodes(*subscription, *timeout, *removeInvalid)
	default:
		return fmt.Errorf("未知节点命令 %q", args[0])
	}
}

func (a *App) printNodes(data state.Data, subscriptionID string) error {
	writer := tabwriter.NewWriter(a.output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "序号\t活动\tID\t名称\t协议\t来源")
	for index, node := range data.Nodes {
		if subscriptionID != "" && node.SubscriptionID != subscriptionID {
			continue
		}
		protocol := strings.SplitN(node.URI, "://", 2)[0]
		active := ""
		if node.ID == data.Settings.ActiveNodeID {
			active = "*"
		}
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\n", index+1, active, node.ID, node.Name, protocol, emptyAs(subscriptionName(data, node.SubscriptionID), "手工"))
	}
	return writer.Flush()
}

func (a *App) testNodes(subscriptionID string, timeout time.Duration, removeInvalid bool) error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	if subscriptionID != "" {
		index, err := findSubscription(data, subscriptionID)
		if err != nil {
			return err
		}
		subscriptionID = data.Subscriptions[index].ID
	}
	binary := data.Settings.XrayPath
	if binary == "" {
		binary = "xray"
	}
	invalid := map[string]bool{}
	count := 0
	for _, node := range data.Nodes {
		if subscriptionID != "" && node.SubscriptionID != subscriptionID {
			continue
		}
		count++
		parsed, parseErr := nodes.Parse(node.URI)
		if parseErr != nil {
			fmt.Fprintf(a.output, "[失败] %s: %v\n", node.Name, parseErr)
			invalid[node.ID] = true
			continue
		}
		latency, testErr := xray.TestOutbound(context.Background(), binary, parsed.Outbound, data.Settings.TestURL, timeout)
		if testErr != nil {
			fmt.Fprintf(a.output, "[失败] %s: %v\n", node.Name, testErr)
			invalid[node.ID] = true
			continue
		}
		fmt.Fprintf(a.output, "[成功] %s: %s\n", node.Name, latency.Round(time.Millisecond))
	}
	if count == 0 {
		return fmt.Errorf("没有可测试的节点")
	}
	if removeInvalid && len(invalid) > 0 {
		data.Nodes = filterNodes(data.Nodes, func(node state.Node) bool { return !invalid[node.ID] })
		normalizeActiveNode(&data)
		if err := a.store.Save(data); err != nil {
			return err
		}
		fmt.Fprintf(a.output, "[成功] 已删除 %d 个失效节点\n", len(invalid))
	}
	return nil
}

func (a *App) service(action string) error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	switch action {
	case "start":
		if xray.PortOpen(data.Settings.ListenPort) {
			return fmt.Errorf("连接已启动，mixed 代理监听于 127.0.0.1:%d", data.Settings.ListenPort)
		}
		index, err := findNode(data, data.Settings.ActiveNodeID)
		if err != nil {
			return fmt.Errorf("尚未选择活动节点，请先运行 node use <ID>")
		}
		parsed, err := nodes.Parse(data.Nodes[index].URI)
		if err != nil {
			return err
		}
		binary := data.Settings.XrayPath
		if binary == "" {
			binary = "xray"
		}
		pid, configPath, err := xray.Start(context.Background(), binary, parsed.Outbound, data.Settings.ListenPort, a.store.RuntimeDir())
		if err != nil {
			return err
		}
		data.Runtime = state.Runtime{PID: pid, NodeID: data.Nodes[index].ID, ConfigPath: configPath, StartedAt: time.Now()}
		if err := a.store.Save(data); err != nil {
			_ = xray.Stop(pid)
			return err
		}
		fmt.Fprintf(a.output, "[成功] 连接已启动: %s，mixed 代理 127.0.0.1:%d，PID %d\n", data.Nodes[index].Name, data.Settings.ListenPort, pid)
		return nil
	case "stop":
		if data.Settings.GlobalProxy {
			if err := systemproxy.Disable(); err != nil {
				return fmt.Errorf("关闭全局代理失败，连接未停止: %w", err)
			}
			data.Settings.GlobalProxy = false
		}
		if !xray.PortOpen(data.Settings.ListenPort) {
			data.Runtime = state.Runtime{}
			if err := a.store.Save(data); err != nil {
				return err
			}
			fmt.Fprintln(a.output, "[信息] 连接未运行")
			return nil
		}
		if err := xray.Stop(data.Runtime.PID); err != nil {
			return err
		}
		data.Runtime = state.Runtime{}
		if err := a.store.Save(data); err != nil {
			return err
		}
		fmt.Fprintln(a.output, "[成功] 连接已停止")
		return nil
	case "status":
		if !xray.PortOpen(data.Settings.ListenPort) {
			fmt.Fprintln(a.output, "状态: stopped")
			return nil
		}
		name := "未知节点"
		if index, findErr := findNode(data, data.Runtime.NodeID); findErr == nil {
			name = data.Nodes[index].Name
		}
		fmt.Fprintf(a.output, "状态: running\n节点: %s\nmixed 代理: 127.0.0.1:%d\nPID: %d\n全局代理: %t\n", name, data.Settings.ListenPort, data.Runtime.PID, data.Settings.GlobalProxy)
		return nil
	default:
		return fmt.Errorf("未知服务操作 %q", action)
	}
}

func (a *App) proxy(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("用法: x-cmd proxy enable|disable|status")
	}
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	switch args[0] {
	case "enable", "on":
		if !xray.PortOpen(data.Settings.ListenPort) {
			return fmt.Errorf("连接尚未启动，请先运行 x-cmd system start")
		}
		if err := systemproxy.Enable(data.Settings.ListenPort); err != nil {
			return fmt.Errorf("开启全局代理失败: %w", err)
		}
		data.Settings.GlobalProxy = true
	case "disable", "off":
		if err := systemproxy.Disable(); err != nil {
			return fmt.Errorf("关闭全局代理失败: %w", err)
		}
		data.Settings.GlobalProxy = false
	case "status":
		fmt.Fprintf(a.output, "全局代理: %t\n地址: %s\n", data.Settings.GlobalProxy, systemproxy.Address(data.Settings.ListenPort))
		return nil
	default:
		return fmt.Errorf("用法: x-cmd proxy enable|disable|status")
	}
	if err := a.store.Save(data); err != nil {
		return err
	}
	fmt.Fprintf(a.output, "全局代理: %t\n", data.Settings.GlobalProxy)
	return nil
}

func (a *App) update(args []string) error {
	action := "check"
	if len(args) > 0 {
		action = args[0]
	}
	if action != "check" && action != "install" {
		return fmt.Errorf("用法: x-cmd update check|install")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	rewriters, err := githuburl.Candidates(data.Settings.GitHubMirror)
	if err != nil {
		return err
	}
	var release updater.Release
	var selected githuburl.Rewriter
	for index, rewriter := range rewriters {
		if index > 0 {
			fmt.Fprintln(a.output, "[提示] GitHub 直连不可用，正在切换到内置镜像...")
		}
		release, err = updater.Latest(ctx, version.Repository, rewriter)
		if err == nil {
			selected = rewriter
			break
		}
	}
	if err != nil {
		return err
	}
	newer := updater.IsNewer(release.TagName, version.Version)
	if !newer {
		fmt.Fprintf(a.output, "[信息] 当前已是最新版本: %s\n", version.Version)
		return nil
	}
	fmt.Fprintf(a.output, "[信息] 发现新版本: %s（当前 %s）\n", release.TagName, version.Version)
	if action == "check" {
		fmt.Fprintln(a.output, "[提示] 运行 x-cmd update install 在线更新")
		return nil
	}
	installCandidates := []githuburl.Rewriter{selected}
	if data.Settings.GitHubMirror == "" && selected.Mirror == "" {
		installCandidates = rewriters
	}
	for index, rewriter := range installCandidates {
		if index > 0 {
			fmt.Fprintln(a.output, "[提示] GitHub 下载不可用，正在切换到内置镜像...")
		}
		err = updater.Install(ctx, release, rewriter)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(a.output, "[成功] 更新完成，请重新启动 x-cmd")
	return nil
}

func (a *App) interactive() error {
	for {
		a.printMenu("X-CMD  Xray 管理工具\n1. 内核信息与安装\n2. 订阅管理\n3. 节点管理\n4. 测试全部节点并清理失效项\n5. 修改配置\n6. 连接管理\n7. 全局代理开关\nu. 卸载\n0. 退出")
		choice := a.prompt("请选择")
		var err error
		switch choice {
		case "1":
			err = a.interactiveCore()
		case "2":
			err = a.interactiveSubscriptions()
		case "3":
			err = a.interactiveNodes()
		case "4":
			err = a.testNodes("", 10*time.Second, strings.EqualFold(a.prompt("确认自动删除失效节点? [y/N]"), "y"))
		case "5":
			err = a.interactiveConfig()
		case "6":
			err = a.interactiveSystem()
		case "7":
			action := "enable"
			data, loadErr := a.store.Load()
			if loadErr != nil {
				err = loadErr
			} else {
				if data.Settings.GlobalProxy {
					action = "disable"
				}
				err = a.proxy([]string{action})
			}
		case "u", "U":
			if strings.EqualFold(a.prompt("卸载会删除程序和全部配置，确认? [y/N]"), "y") {
				return a.uninstallApp([]string{"--yes"})
			}
		case "0":
			return nil
		default:
			fmt.Fprintln(a.output, "[提示] 无效选项，请重新输入")
		}
		if err != nil {
			fmt.Fprintln(a.output, "[错误]", err)
		}
		a.waitForMenu(2)
	}
}

func (a *App) waitForMenu(seconds int) {
	for remaining := seconds; remaining > 0; remaining-- {
		fmt.Fprintf(a.output, "[提示] %d 秒后返回\n", remaining)
		if a.pause != nil {
			a.pause(time.Second)
		}
	}
}

func (a *App) interactiveSystem() error {
	a.printMenu("1. 启动连接  2. 查看状态  3. 停止连接  0. 返回")
	action := map[string]string{"1": "start", "2": "status", "3": "stop"}[a.prompt("操作")]
	if action == "" {
		return nil
	}
	return a.system([]string{action})
}

func (a *App) interactiveCore() error {
	a.printMenu("1. 查看当前内核信息  2. 查看最近 Release  3. 安装/切换内核  0. 返回")
	switch a.prompt("操作") {
	case "1":
		return a.core([]string{"show"})
	case "2":
		return a.core([]string{"releases"})
	case "3":
		version := a.prompt("版本（例如 v26.3.27）")
		return a.core([]string{"install", "--version", version})
	default:
		return nil
	}
}

func (a *App) interactiveSubscriptions() error {
	if err := a.listSubscriptions(); err != nil {
		return err
	}
	a.printMenu("a. 添加  e. 编辑  d. 删除  u. 更新  n. 查看节点  0. 返回")
	switch strings.ToLower(a.prompt("操作")) {
	case "a":
		return a.subscriptions([]string{"add", "--name", a.prompt("名称"), "--url", a.prompt("链接")})
	case "e":
		id := a.prompt("订阅 ID")
		name := a.prompt("新名称（留空则不改）")
		address := a.prompt("新链接（留空则不改）")
		args := []string{"edit", id}
		if name != "" {
			args = append(args, "--name", name)
		}
		if address != "" {
			args = append(args, "--url", address)
		}
		return a.subscriptions(args)
	case "d":
		return a.subscriptions([]string{"delete", a.prompt("订阅 ID")})
	case "u":
		return a.subscriptions([]string{"update", defaultString(a.prompt("订阅 ID（留空更新全部）"), "all")})
	case "n":
		return a.subscriptions([]string{"nodes", a.prompt("订阅 ID")})
	}
	return nil
}

func (a *App) interactiveNodes() error {
	data, err := a.store.Load()
	if err != nil {
		return err
	}
	if err := a.printNodes(data, ""); err != nil {
		return err
	}
	a.printMenu("s. 选择  a. 添加  d. 删除  t. 测试  0. 返回")
	switch strings.ToLower(a.prompt("操作")) {
	case "s":
		return a.node([]string{"use", a.prompt("节点序号或 ID")})
	case "a":
		return a.node([]string{"add", "--uri", a.prompt("节点分享链接"), "--name", a.prompt("名称（可留空）")})
	case "d":
		return a.node([]string{"delete", a.prompt("节点 ID")})
	case "t":
		return a.testNodes("", 10*time.Second, false)
	}
	return nil
}

func (a *App) interactiveConfig() error {
	if err := a.config([]string{"show"}); err != nil {
		return err
	}
	a.printMenu("1. 下载地址  2. Xray 路径  3. 测试地址  4. GitHub 镜像  0. 返回")
	choice := a.prompt("配置项")
	key := map[string]string{"1": "--download-url", "2": "--xray-path", "3": "--test-url", "4": "--github-mirror"}[choice]
	if key == "" {
		return nil
	}
	return a.config([]string{"set", key, a.prompt("新值")})
}

func (a *App) prompt(label string) string {
	fmt.Fprint(a.output, label+": ")
	value, _ := a.input.ReadString('\n')
	return strings.TrimSpace(value)
}

func (a *App) printMenu(menu string) {
	fmt.Fprintf(a.output, "\n\x1b[1;96m%s\x1b[0m\n", menu)
}

func (a *App) printHelp() {
	fmt.Fprintln(a.output, `x-cmd - xray-core 命令行管理工具

无参数启动中文交互菜单。

命令:
	system start|status|stop
	proxy enable|disable|status
	update check|install
	uninstall [--yes]
	completion install|uninstall [bash|zsh|fish|powershell]
	github-mirror show|set <URL>|delete
	core show|releases|install --version VERSION [--dir DIR]
	config show|set [--download-url URL] [--github-mirror URL] [--xray-path PATH] [--test-url URL] [--listen-port PORT]
  sub list|add|edit|delete|update|nodes
	node list|add|use|delete|test [--delete-invalid] [--subscription ID]
  version

运行 x-cmd <命令> -h 查看参数。`)
}

func newFlagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	return set
}
func emptyAs(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04")
}
func filterNodes(values []state.Node, keep func(state.Node) bool) []state.Node {
	result := values[:0]
	for _, value := range values {
		if keep(value) {
			result = append(result, value)
		}
	}
	return result
}

func normalizeActiveNode(data *state.Data) {
	for _, node := range data.Nodes {
		if node.ID == data.Settings.ActiveNodeID {
			return
		}
	}
	data.Settings.ActiveNodeID = ""
	if len(data.Nodes) > 0 {
		data.Settings.ActiveNodeID = data.Nodes[0].ID
	}
}

func findSubscription(data state.Data, prefix string) (int, error) {
	matches := []int{}
	for index, subscription := range data.Subscriptions {
		if strings.HasPrefix(subscription.ID, prefix) {
			matches = append(matches, index)
		}
	}
	if len(matches) != 1 {
		return -1, fmt.Errorf("订阅 ID %q 匹配到 %d 项", prefix, len(matches))
	}
	return matches[0], nil
}

func findNode(data state.Data, prefix string) (int, error) {
	matches := []int{}
	for index, node := range data.Nodes {
		if strings.HasPrefix(node.ID, prefix) {
			matches = append(matches, index)
		}
	}
	if len(matches) != 1 {
		return -1, fmt.Errorf("节点 ID %q 匹配到 %d 项", prefix, len(matches))
	}
	return matches[0], nil
}

func findNodeSelection(data state.Data, selection string) (int, error) {
	if number, err := strconv.Atoi(selection); err == nil {
		if number < 1 || number > len(data.Nodes) {
			return -1, fmt.Errorf("节点序号 %d 超出范围 1-%d", number, len(data.Nodes))
		}
		return number - 1, nil
	}
	return findNode(data, selection)
}

func subscriptionName(data state.Data, id string) string {
	for _, subscription := range data.Subscriptions {
		if subscription.ID == id {
			return subscription.Name
		}
	}
	return ""
}
