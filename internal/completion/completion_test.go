package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidates(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"sys"}, "system"},
		{[]string{"system", "st"}, "start\nstatus\nstop"},
		{[]string{"uninstall", "-"}, "--yes"},
		{[]string{"completion", "install", "p"}, "powershell"},
	}
	for _, test := range tests {
		if got := strings.Join(Candidates(test.args), "\n"); got != test.want {
			t.Fatalf("Candidates(%q) = %q, want %q", test.args, got, test.want)
		}
	}
}

func TestInstallAndUninstallBash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	paths, err := Install("bash")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("Install returned %d paths", len(paths))
	}
	profile := filepath.Join(home, ".bashrc")
	raw, err := os.ReadFile(profile)
	if err != nil || !strings.Contains(string(raw), markerStart) {
		t.Fatalf("completion marker missing: %q, %v", raw, err)
	}
	if err := Uninstall("bash"); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), markerStart) {
		t.Fatalf("completion marker was not removed: %q", raw)
	}
}
