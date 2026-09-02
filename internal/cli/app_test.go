package cli

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"
	"testing"

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

func TestNewAlwaysUsesDefaultConfigPath(t *testing.T) {
	t.Setenv("X_CMD_CONFIG", filepath.Join(t.TempDir(), "custom.json"))
	want, err := state.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if got := New().store.Path(); got != want {
		t.Fatalf("config path = %q, want %q", got, want)
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
	var output bytes.Buffer
	var configPath string
	var runtimeDir string
	app := &App{
		store:  state.New(filepath.Join(t.TempDir(), "config.json")),
		input:  bufio.NewReader(strings.NewReader("")),
		output: &output,
		uninstall: func(config, runtime string) error {
			configPath = config
			runtimeDir = runtime
			return nil
		},
	}
	if err := app.Run([]string{"uninstall"}); err == nil {
		t.Fatal("expected confirmation error")
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
}
