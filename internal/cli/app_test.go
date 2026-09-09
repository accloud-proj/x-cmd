package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/accloud-proj/x-cmd/internal/githuburl"
	"github.com/accloud-proj/x-cmd/internal/state"
	"github.com/accloud-proj/x-cmd/internal/version"
)

func TestVersionFlag(t *testing.T) {
	var output bytes.Buffer
	app := &App{output: &output}
	if err := app.Run([]string{"-v"}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "x-cmd "+version.Version+"\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWaitForMenuWaitsForKey(t *testing.T) {
	var output bytes.Buffer
	waited := false
	app := &App{
		output:  &output,
		waitKey: func() error { waited = true; return nil },
	}
	app.waitForMenu()
	if got := output.String(); got != "\n[提示] 按任意键返回" {
		t.Fatalf("unexpected countdown output: %q", got)
	}
	if !waited {
		t.Fatal("wait key was not called")
	}
}

func TestWriteTableAlignsChineseText(t *testing.T) {
	var output bytes.Buffer
	writeTable(&output, [][]string{{"名称", "协议"}, {"香港", "vless"}, {"US", "trojan"}})
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	for index, line := range lines {
		if position := strings.Index(line, map[int]string{0: "协议", 1: "vless", 2: "trojan"}[index]); displayWidth(line[:position]) != 6 {
			t.Fatalf("line %d is not aligned: %q", index, line)
		}
	}
}

func TestFindSubscriptionByNumberOrName(t *testing.T) {
	data := state.Data{Subscriptions: []state.Subscription{
		{ID: "random-first", Name: "工作"},
		{ID: "random-second", Name: "Home"},
	}}
	for selection, want := range map[string]int{"1": 0, "2": 1, "工作": 0, "home": 1} {
		got, err := findSubscription(data, selection)
		if err != nil {
			t.Fatalf("findSubscription(%q): %v", selection, err)
		}
		if got != want {
			t.Fatalf("findSubscription(%q) = %d, want %d", selection, got, want)
		}
	}
}

func TestFindSubscriptionRejectsDuplicateName(t *testing.T) {
	data := state.Data{Subscriptions: []state.Subscription{
		{ID: "first", Name: "工作"},
		{ID: "second", Name: "工作"},
	}}
	if _, err := findSubscription(data, "工作"); err == nil || !strings.Contains(err.Error(), "请使用序号") {
		t.Fatalf("expected duplicate-name error, got %v", err)
	}
}

func TestNodeSelectionStaysInSubscription(t *testing.T) {
	data := state.Data{Nodes: []state.Node{
		{ID: "first", SubscriptionID: "sub-a"},
		{ID: "second", SubscriptionID: "sub-b"},
	}}
	if got, err := nodeSelectionInSubscription(data, "1", "sub-a"); err != nil || got != 0 {
		t.Fatalf("expected first node, got index %d, error %v", got, err)
	}
	if _, err := nodeSelectionInSubscription(data, "2", "sub-a"); err == nil {
		t.Fatal("node from another subscription should be rejected")
	}
}

func TestConfigPathCommandIsRemoved(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		output: &output,
	}
	if err := app.Run([]string{"config", "path", "show"}); err == nil {
		t.Fatal("config path command should be rejected")
	}
	if err := app.Run([]string{"config", "show"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "config-path") {
		t.Fatalf("config path should not be displayed: %q", output.String())
	}
}

func TestSystemCommandGroupsServiceActions(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		output: &output,
	}
	if err := app.Run([]string{"system", "status"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "stopped") {
		t.Fatalf("unexpected status output: %s", output.String())
	}
	if err := app.Run([]string{"status"}); err == nil {
		t.Fatal("top-level status should be rejected")
	}
}

func TestActivateOutputsShellProxyConfiguration(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	data.Runtime.PID = 42
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	app := &App{store: store, output: &output, portOpen: func(int) bool { return true }, processRunning: func(int) bool { return true }}
	if err := app.Run([]string{"activate", "bash"}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"export HTTP_PROXY=http://127.0.0.1:1091",
		"export HTTPS_PROXY=http://127.0.0.1:1091",
		"export ALL_PROXY=socks5://127.0.0.1:1091",
		"export NO_PROXY=localhost,127.0.0.1,::1",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("activate output missing %q: %q", expected, output.String())
		}
	}
}

func TestActivateAndShellRequireRunningProxy(t *testing.T) {
	app := &App{
		store:    state.New(filepath.Join(t.TempDir(), "config.json")),
		output:   io.Discard,
		portOpen: func(int) bool { return false },
		runShell: func(string, []string, []string) error { t.Fatal("shell must not start"); return nil },
	}
	for _, command := range []string{"activate", "shell"} {
		if err := app.Run([]string{command, "pwsh"}); err == nil || !strings.Contains(err.Error(), "连接尚未启动") {
			t.Fatalf("%s should require a running proxy, got %v", command, err)
		}
	}
}

func TestShellInheritsEnvironmentAndAddsProxy(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	data.Runtime.PID = 42
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	t.Setenv("X_CMD_INHERITED", "yes")
	var executable string
	var args []string
	var environment []string
	app := &App{store: store, output: io.Discard, portOpen: func(int) bool { return true }, processRunning: func(int) bool { return true }}
	app.runShell = func(name string, commandArgs, env []string) error {
		executable, args, environment = name, commandArgs, env
		return nil
	}
	if err := app.Run([]string{"shell", "powershell"}); err != nil {
		t.Fatal(err)
	}
	if executable != "powershell" || !reflect.DeepEqual(args, []string{"-NoExit"}) {
		t.Fatalf("shell command = %q %#v", executable, args)
	}
	joined := strings.Join(environment, "\n")
	for _, expected := range []string{"X_CMD_INHERITED=yes", "HTTP_PROXY=http://127.0.0.1:1091", "ALL_PROXY=socks5://127.0.0.1:1091"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("shell environment missing %q", expected)
		}
	}
}

func TestStatusRejectsOpenPortWhenRecordedProcessStopped(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	data.Runtime.PID = 42
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	app := &App{
		store:          store,
		output:         &output,
		portOpen:       func(int) bool { return true },
		processRunning: func(int) bool { return false },
	}
	if err := app.Run([]string{"system", "status"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "状态: stopped") {
		t.Fatalf("unexpected status: %q", output.String())
	}
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Runtime.PID != 0 {
		t.Fatalf("stale runtime PID was not cleared: %d", saved.Runtime.PID)
	}
}

func TestInteractiveSystemRefreshesOnEnterAndContainsNodeTest(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("\n0\n")),
		output: &output,
	}
	if err := app.interactiveSystem(); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(output.String(), "4. 测试全部节点"); count != 2 {
		t.Fatalf("connection menu shown %d times, want 2: %q", count, output.String())
	}
}

func TestInteractiveSystemWaitsAfterInvalidChoice(t *testing.T) {
	var output bytes.Buffer
	waits := 0
	app := &App{
		store:   state.New(filepath.Join(t.TempDir(), "config.json")),
		input:   bufio.NewReader(strings.NewReader("invalid\n0\n")),
		output:  &output,
		waitKey: func() error { waits++; return nil },
	}
	if err := app.interactiveSystem(); err != nil {
		t.Fatal(err)
	}
	if waits != 1 || !strings.Contains(output.String(), "按任意键返回") {
		t.Fatalf("invalid choice waits = %d, output = %q", waits, output.String())
	}
}

func TestMainMenuIsFullScreenAndContainsProxyShellActions(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		input:  bufio.NewReader(strings.NewReader("0\n")),
		output: &output,
	}
	if err := app.interactive(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\x1b[2J\x1b[H", "当前版本: " + version.Version, "6. 进入 Shell", "7. 查看临时激活命令"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("main menu missing %q: %q", expected, output.String())
		}
	}
	if strings.Contains(output.String(), "4. 测试全部节点") {
		t.Fatalf("node test should not remain in main menu: %q", output.String())
	}
	for _, expected := range []string{"\x1b[2J\x1b[H  \x1b[1;96mX-CMD", "\n  当前版本:", "\n  1. 内核信息与安装", "\n  0. 退出"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("main menu is not indented: missing %q in %q", expected, output.String())
		}
	}
}

func TestMainMenuOpensMaintenanceSubmenu(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		input:  bufio.NewReader(strings.NewReader("u\n0\n0\n")),
		output: &output,
	}
	if err := app.interactive(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"u. 更新/卸载", "1. 检测更新  2. 更新  3. 卸载  0. 返回"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("maintenance menu missing %q: %q", expected, output.String())
		}
	}
}

func TestInteractiveMaintenanceUninstallsAndExits(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	removed := false
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("3\ny\n")),
		output: io.Discard,
		uninstall: func(string, string) error {
			removed = true
			return nil
		},
	}
	uninstalled, err := app.interactiveMaintenance()
	if err != nil {
		t.Fatal(err)
	}
	if !uninstalled || !removed {
		t.Fatalf("uninstalled = %t, removed = %t", uninstalled, removed)
	}
}

func TestInteractiveMessagesAreIndented(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		input:   bufio.NewReader(strings.NewReader("invalid\n0\n")),
		output:  &output,
		waitKey: func() error { return nil },
	}
	if err := app.interactive(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "  请选择:   [提示] 无效选项") || !strings.Contains(output.String(), "\n  [提示] 按任意键返回") {
		t.Fatalf("interactive messages are not indented: %q", output.String())
	}
}

func TestClearScreenPrintsBannerAtTop(t *testing.T) {
	var output bytes.Buffer
	app := &App{output: &output, banner: "BANNER\n"}
	app.clearScreen()
	if got, want := output.String(), "\x1b[2J\x1b[HBANNER\n"; got != want {
		t.Fatalf("clear output = %q, want %q", got, want)
	}
}

func TestInteractiveConfigChangesListenPort(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	app := &App{
		store:   store,
		input:   bufio.NewReader(strings.NewReader("5\n2080\n0\n")),
		output:  io.Discard,
		waitKey: func() error { return nil },
	}
	if err := app.interactiveConfig(); err != nil {
		t.Fatal(err)
	}
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.ListenPort != 2080 {
		t.Fatalf("listen port = %d, want 2080", data.Settings.ListenPort)
	}
}

func TestInteractiveConfigEnablesLANAccess(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	app := &App{
		store:   store,
		input:   bufio.NewReader(strings.NewReader("6\ny\n0\n")),
		output:  io.Discard,
		waitKey: func() error { return nil },
	}
	if err := app.interactiveConfig(); err != nil {
		t.Fatal(err)
	}
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !data.Settings.AllowLAN {
		t.Fatal("LAN access was not enabled")
	}
}

func TestInteractiveConfigDisablesLANAccess(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Settings.AllowLAN = true
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	app := &App{
		store:   store,
		input:   bufio.NewReader(strings.NewReader("6\nn\n0\n")),
		output:  io.Discard,
		waitKey: func() error { return nil },
	}
	if err := app.interactiveConfig(); err != nil {
		t.Fatal(err)
	}
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.AllowLAN {
		t.Fatal("LAN access was not disabled")
	}
}

func TestMenuProxyActionsUseAutomaticShellDetection(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	var output bytes.Buffer
	detected := 0
	app := &App{
		store:    store,
		input:    bufio.NewReader(strings.NewReader("6\n7\n0\n")),
		output:   &output,
		portOpen: func(int) bool { return true },
		waitKey:  func() error { return nil },
		detectShell: func() (string, error) {
			detected++
			return "pwsh", nil
		},
		runShell: func(name string, _ []string, _ []string) error {
			if name != "pwsh" {
				t.Fatalf("shell = %q, want pwsh", name)
			}
			return nil
		},
	}
	if err := app.interactive(); err != nil {
		t.Fatal(err)
	}
	if detected != 2 || !strings.Contains(output.String(), "自动识别 Shell: pwsh") {
		t.Fatalf("detected = %d, output = %q", detected, output.String())
	}
}

func TestReturningFromNestedNodeMenuDoesNotWait(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Subscriptions = []state.Subscription{{ID: "sub", Name: "example"}}
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	waits := 0
	app := &App{
		store:   store,
		input:   bufio.NewReader(strings.NewReader("n\n1\n0\n0\n")),
		output:  io.Discard,
		waitKey: func() error { waits++; return nil },
	}
	if err := app.interactiveSubscriptions(); err != nil {
		t.Fatal(err)
	}
	if waits != 0 {
		t.Fatalf("returning with 0 waited %d times", waits)
	}
}

func TestNodeAddAndList(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("")),
		output: &output,
	}
	link := "trojan://secret@example.com:443?security=tls#example"
	if err := app.Run([]string{"node", "add", "--uri", link}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"node", "list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "example") || !strings.Contains(output.String(), "trojan") {
		t.Fatalf("unexpected output: %s", output.String())
	}
	data, err := app.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Nodes) != 1 || data.Settings.ActiveNodeID != data.Nodes[0].ID {
		t.Fatalf("first node was not selected: %#v", data.Settings)
	}
}

func TestNodeListAndUseAcceptNumberOrID(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("")),
		output: &output,
	}
	for _, link := range []string{
		"trojan://secret@one.example:443?security=tls#one",
		"trojan://secret@two.example:443?security=tls#two",
	} {
		if err := app.Run([]string{"node", "add", "--uri", link}); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Run([]string{"node", "list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "序号") || !strings.Contains(output.String(), "2") {
		t.Fatalf("node numbers missing from output: %s", output.String())
	}
	if err := app.Run([]string{"node", "use", "2"}); err != nil {
		t.Fatal(err)
	}
	data, err := app.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.ActiveNodeID != data.Nodes[1].ID {
		t.Fatalf("number selected %q, want %q", data.Settings.ActiveNodeID, data.Nodes[1].ID)
	}
	if err := app.Run([]string{"node", "use", data.Nodes[0].ID}); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteActiveNodeSelectsNextAndStopsConnection(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Nodes = []state.Node{
		{ID: "one", Name: "one"},
		{ID: "two", Name: "two"},
		{ID: "three", Name: "three"},
	}
	data.Settings.ActiveNodeID = "two"
	data.Runtime = state.Runtime{PID: 42, NodeID: "two"}
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	var actions []string
	app := &App{store: store, output: io.Discard}
	app.serviceFn = func(action string) error {
		actions = append(actions, action)
		current, err := store.Load()
		if err != nil {
			return err
		}
		current.Runtime = state.Runtime{}
		return store.Save(current)
	}
	if err := app.Run([]string{"node", "delete", "2"}); err != nil {
		t.Fatal(err)
	}
	data, _ = store.Load()
	if len(data.Nodes) != 2 || data.Settings.ActiveNodeID != "three" {
		t.Fatalf("unexpected nodes after delete: %#v, active %q", data.Nodes, data.Settings.ActiveNodeID)
	}
	if strings.Join(actions, ",") != "stop" {
		t.Fatalf("service actions = %v", actions)
	}
}

func TestDeleteOnlyActiveNodeClearsSelection(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Nodes = []state.Node{{ID: "one", Name: "one"}}
	data.Settings.ActiveNodeID = "one"
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	app := &App{store: store, output: io.Discard}
	if err := app.Run([]string{"node", "delete", "one"}); err != nil {
		t.Fatal(err)
	}
	data, _ = store.Load()
	if len(data.Nodes) != 0 || data.Settings.ActiveNodeID != "" {
		t.Fatalf("unexpected state: %#v", data)
	}
}

func TestSwitchRunningNodeRestartsConnection(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Nodes = []state.Node{{ID: "one", Name: "one"}, {ID: "two", Name: "two"}}
	data.Settings.ActiveNodeID = "one"
	data.Runtime = state.Runtime{PID: 42, NodeID: "one"}
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	var actions []string
	app := &App{store: store, output: io.Discard}
	app.serviceFn = func(action string) error {
		actions = append(actions, action)
		current, err := store.Load()
		if err != nil {
			return err
		}
		if action == "stop" {
			current.Runtime = state.Runtime{}
		}
		return store.Save(current)
	}
	if err := app.Run([]string{"node", "use", "2"}); err != nil {
		t.Fatal(err)
	}
	data, _ = store.Load()
	if data.Settings.ActiveNodeID != "two" || strings.Join(actions, ",") != "stop,start" {
		t.Fatalf("active = %q, actions = %v", data.Settings.ActiveNodeID, actions)
	}
}

func TestInteractiveNodeActionReturnsToNodeMenu(t *testing.T) {
	store := state.New(filepath.Join(t.TempDir(), "config.json"))
	data, _ := store.Load()
	data.Nodes = []state.Node{{ID: "one", Name: "one", URI: "trojan://secret@example.com:443?security=tls#one"}}
	data.Settings.ActiveNodeID = "one"
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	app := &App{
		store:   store,
		input:   bufio.NewReader(strings.NewReader("s\n1\n0\n")),
		output:  &output,
		waitKey: func() error { return nil },
	}
	if err := app.interactiveNodes(""); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(output.String(), "s. 选择"); count != 2 {
		t.Fatalf("node menu shown %d times, want 2: %q", count, output.String())
	}
}

func TestGitHubMirrorLifecycle(t *testing.T) {
	var output bytes.Buffer
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("")),
		output: &output,
	}
	if err := app.Run([]string{"github-mirror", "set", "https://mirror.example/"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"github-mirror", "show"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "https://mirror.example") {
		t.Fatalf("unexpected output: %s", output.String())
	}
	if err := app.Run([]string{"github-mirror", "delete"}); err != nil {
		t.Fatal(err)
	}
	data, err := app.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.GitHubMirror != "" {
		t.Fatalf("mirror was not deleted: %q", data.Settings.GitHubMirror)
	}
}

func TestSlowGitHubDirectConnectionPrefersBuiltInMirror(t *testing.T) {
	var output bytes.Buffer
	var minimumBytesPerSecond int64
	app := &App{
		output: &output,
		githubProbe: func(_ context.Context, _ string, minimum int64) (bool, float64, error) {
			minimumBytesPerSecond = minimum
			return false, 6250, nil
		},
	}
	candidates, err := app.githubCandidates(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Mirror != githuburl.DefaultMirror {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
	if minimumBytesPerSecond != 12500 {
		t.Fatalf("minimum speed = %d bytes/s, want 12500", minimumBytesPerSecond)
	}
	if !strings.Contains(output.String(), "低于 100 kbit/s") {
		t.Fatalf("missing slow connection notice: %q", output.String())
	}
}

func TestConfiguredGitHubMirrorSkipsSpeedTest(t *testing.T) {
	for _, mirror := range []string{"https://mirror.example", githuburl.DefaultMirror} {
		called := false
		app := &App{
			output: io.Discard,
			githubProbe: func(context.Context, string, int64) (bool, float64, error) {
				called = true
				return false, 0, nil
			},
		}
		candidates, err := app.githubCandidates(context.Background(), mirror)
		if err != nil {
			t.Fatal(err)
		}
		if called {
			t.Fatalf("configured mirror %q must skip the GitHub speed test", mirror)
		}
		if len(candidates) != 1 || candidates[0].Mirror != mirror {
			t.Fatalf("configured mirror %q changed: %#v", mirror, candidates)
		}
	}
}

func TestAutomaticallySelectedMirrorBecomesFixedConfiguration(t *testing.T) {
	data := state.Data{}
	app := &App{output: io.Discard}
	if !app.persistSelectedMirror(&data, githuburl.Rewriter{Mirror: githuburl.DefaultMirror}) {
		t.Fatal("automatically selected mirror was not persisted")
	}
	called := false
	app.githubProbe = func(context.Context, string, int64) (bool, float64, error) {
		called = true
		return true, 0, nil
	}
	candidates, err := app.githubCandidates(context.Background(), data.Settings.GitHubMirror)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("persisted automatic mirror must skip subsequent detection")
	}
	if len(candidates) != 1 || candidates[0].Mirror != githuburl.DefaultMirror {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}

func TestUninstallRequiresConfirmationAndRemovesData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	var output bytes.Buffer
	var configPath string
	var runtimeDir string
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("\n")),
		output: &output,
		uninstall: func(config, runtime string) error {
			configPath = config
			runtimeDir = runtime
			return nil
		},
	}
	if err := app.Run([]string{"completion", "install", "bash"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"uninstall"}); err != nil {
		t.Fatal(err)
	}
	if configPath != "" || !strings.Contains(output.String(), "已取消卸载") {
		t.Fatalf("default confirmation should cancel: %q", output.String())
	}
	if err := app.Run([]string{"uninstall", "--yes"}); err != nil {
		t.Fatal(err)
	}
	if configPath != app.store.Path() || runtimeDir != app.store.RuntimeDir() {
		t.Fatalf("unexpected uninstall paths: %q %q", configPath, runtimeDir)
	}
	if !strings.Contains(output.String(), "卸载完成") {
		t.Fatalf("unexpected output: %s", output.String())
	}
	profile, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(profile), "x-cmd completion") {
		t.Fatalf("completion was not removed: %q", profile)
	}
}

func TestUninstallAcceptsInteractiveConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	removed := false
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("yes\n")),
		output: io.Discard,
		uninstall: func(string, string) error {
			removed = true
			return nil
		},
	}
	if err := app.Run([]string{"uninstall"}); err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("interactive confirmation did not uninstall")
	}
}

func TestCompletionCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	var output bytes.Buffer
	app := &App{output: &output}
	if err := app.Run([]string{"completion", "install", "bash"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "命令补全安装完成") {
		t.Fatalf("unexpected output: %q", output.String())
	}
	if err := app.Run([]string{"completion", "candidates", "system", "st"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "status") {
		t.Fatalf("completion candidates missing: %q", output.String())
	}
	if err := app.Run([]string{"completion", "uninstall", "bash"}); err != nil {
		t.Fatal(err)
	}
}
