package shellenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestEnvironmentForShellMarksProxyPrompt(t *testing.T) {
	for shell, key := range map[string]string{"sh": "PS1=", "cmd": "PROMPT="} {
		environment := EnvironmentForShell(nil, 1091, shell)
		joined := strings.Join(environment, "\n")
		if !strings.Contains(joined, key) || !strings.Contains(joined, "[proxy]") {
			t.Errorf("prompt for %s is missing from environment: %q", shell, joined)
		}
	}
}

func TestPreparePromptWrapsBashStartupAfterUserConfig(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export TEST_BASHRC=loaded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args, _, cleanup, err := preparePrompt("bash", []string{"-i"}, []string{"HOME=" + home})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(args) != 3 || args[0] != "--rcfile" {
		t.Fatalf("bash args = %#v", args)
	}
	content, err := os.ReadFile(args[1])
	if err != nil {
		t.Fatal(err)
	}
	rc := string(content)
	if !strings.Contains(rc, filepath.Join(home, ".bashrc")) || !strings.Contains(rc, "[proxy]") || strings.Index(rc, "[proxy]") < strings.Index(rc, ".bashrc") {
		t.Fatalf("Bash startup does not apply the marker after the user's config: %s", rc)
	}
}

func TestBashPromptAppearsAfterInteractiveStartup(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("PS1=$'\\noriginal> '\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args, environment, cleanup, err := preparePrompt("bash", []string{"-i", "-c", `__x_cmd_proxy_prompt; printf '%s' "$PS1"`}, []string{"HOME=" + home})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	command := exec.Command(bash, args...)
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("bash failed: %v: %s", err, output)
	}
	prompt := string(output)
	if !strings.Contains(prompt, "\n\\[\\e[1;36m\\][proxy]") || !strings.Contains(prompt, "original> ") || strings.Contains(prompt, "[proxy]\n") {
		t.Fatalf("Bash did not preserve its configured prompt with the proxy marker: %q", output)
	}
}

func TestPreparePromptWrapsZshStartupAndRestoresUserConfig(t *testing.T) {
	home := t.TempDir()
	args, environment, cleanup, err := preparePrompt("zsh", []string{"-i"}, []string{"HOME=" + home})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(args) != 1 || args[0] != "-i" {
		t.Fatalf("zsh args = %#v", args)
	}
	zdotdir := environmentValue(environment, "ZDOTDIR")
	if zdotdir == "" {
		t.Fatal("temporary ZDOTDIR was not set")
	}
	zshenv, err := os.ReadFile(filepath.Join(zdotdir, ".zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	zshrc, err := os.ReadFile(filepath.Join(zdotdir, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(zshenv), filepath.Join(home, ".zshenv")) || !strings.Contains(string(zshrc), filepath.Join(home, ".zshrc")) || !strings.Contains(string(zshrc), "[proxy]") {
		t.Fatalf("Zsh startup does not preserve user config and apply the marker: %s\n%s", zshenv, zshrc)
	}
}

func TestZshPromptAppearsAfterInteractiveStartup(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	home := t.TempDir()
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "starship"), []byte("#!/bin/sh\nprintf '\\nx-cmd metadata\\n❯ %s' \"$*\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("PROMPT='$(starship prompt --status=$?)'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	environmentInput := []string{"HOME=" + home, "PATH=" + bin}
	args, environment, cleanup, err := preparePrompt("zsh", []string{"-i", "-c", `false; __x_cmd_render_proxy_prompt`}, environmentInput)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	command := exec.Command(zsh, args...)
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh failed: %v: %s", err, output)
	}
	prompt := string(output)
	if !strings.Contains(prompt, "%F{cyan}[proxy]%f x-cmd metadata\n❯ prompt --status=1") || strings.Contains(prompt, "[proxy]\n") {
		t.Fatalf("Zsh did not put the proxy marker on Starship's first visible prompt line: %q", output)
	}
}

func TestFishCommandDefinesVisiblePrompt(t *testing.T) {
	_, args, err := Command("fish")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "function fish_prompt") || !strings.Contains(joined, "[proxy]") {
		t.Fatalf("fish command does not define the proxy prompt: %#v", args)
	}
}
