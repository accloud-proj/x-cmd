package uninstaller

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveData(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	runtimeDir := filepath.Join(directory, "runtime")
	if err := os.Mkdir(runtimeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{configPath, configPath + ".tmp", filepath.Join(runtimeDir, "xray.json")} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := removeData(configPath, runtimeDir); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{configPath, configPath + ".tmp", runtimeDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path was not removed: %s", path)
		}
	}
}
