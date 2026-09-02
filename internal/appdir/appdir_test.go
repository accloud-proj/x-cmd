package appdir

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	got, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".local", "x-cmd")
	if runtime.GOOS == "windows" {
		want = filepath.Join(home, "x-cmd")
	}
	if got != want {
		t.Fatalf("Default() = %q, want %q", got, want)
	}
}
