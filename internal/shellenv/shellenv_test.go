package shellenv

import "testing"

func TestDetectPowerShellCoreOnWindows(t *testing.T) {
	environment := map[string]string{
		"COMSPEC":      `C:\Windows\System32\cmd.exe`,
		"PSModulePath": `C:\Users\test\Documents\PowerShell\Modules;C:\Program Files\PowerShell\7\Modules`,
	}
	got := detect("windows", func(key string) string { return environment[key] }, "go.exe")
	if got != "pwsh" {
		t.Fatalf("detect() = %q, want pwsh", got)
	}
}

func TestParentShellTakesPriorityOnWindows(t *testing.T) {
	getenv := func(key string) string {
		if key == "PSModulePath" {
			return `C:\Program Files\PowerShell\7\Modules`
		}
		return ""
	}
	if got := detect("windows", getenv, `C:\Windows\System32\cmd.exe`); got != "cmd" {
		t.Fatalf("detect() = %q, want cmd", got)
	}
	if got := detect("windows", getenv, `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`); got != "powershell" {
		t.Fatalf("detect() = %q, want powershell", got)
	}
}

func TestDetectFallsBackWhenShellInformationIsEmpty(t *testing.T) {
	emptyEnvironment := func(string) string { return "" }
	if got := detect("linux", emptyEnvironment, ""); got != "sh" {
		t.Fatalf("Linux fallback = %q, want sh", got)
	}
	if got := detect("windows", emptyEnvironment, ""); got != "powershell" {
		t.Fatalf("Windows fallback = %q, want powershell", got)
	}
}

func TestShellNamePreservesEmptyValue(t *testing.T) {
	if got := shellName(""); got != "" {
		t.Fatalf("shellName(\"\") = %q, want empty", got)
	}
}
