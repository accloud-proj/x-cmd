package cli

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestWaitForMenuShowsCountdown(t *testing.T) {
	var output bytes.Buffer
	var pauses []time.Duration
	app := &App{
		output: &output,
		pause:  func(duration time.Duration) { pauses = append(pauses, duration) },
	}
	app.waitForMenu(2)
	if got := output.String(); got != "\n[提示] 2 秒后返回\n" {
		t.Fatalf("unexpected countdown output: %q", got)
	}
	if len(pauses) != 1 || pauses[0] != 2*time.Second {
		t.Fatalf("unexpected pauses: %#v", pauses)
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
		store:  store,
		input:  bufio.NewReader(strings.NewReader("s\n1\n0\n")),
		output: &output,
		pause:  func(time.Duration) {},
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
