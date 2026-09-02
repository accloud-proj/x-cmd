//go:build linux

package systemproxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateProxyProfileLifecycle(t *testing.T) {
	profile := filepath.Join(t.TempDir(), ".bashrc")
	if err := os.WriteFile(profile, []byte("export KEEP=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := updateProxyProfile(profile, "export HTTP_PROXY=http://127.0.0.1:1091"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "export KEEP=value") || !strings.Contains(content, proxyMarkerStart) || !strings.Contains(content, "HTTP_PROXY") {
		t.Fatalf("proxy block was not added correctly: %q", content)
	}
	if err := updateProxyProfile(profile, ""); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != "export KEEP=value\n" {
		t.Fatalf("proxy block was not removed cleanly: %q", got)
	}
}

func TestRemoveProxyBlockLeavesUnterminatedBlockUntouched(t *testing.T) {
	content := "keep\n" + proxyMarkerStart + "\nvalue\n"
	if got := removeProxyBlock(content); got != content {
		t.Fatalf("unterminated block changed: %q", got)
	}
}
