package completion

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/accloud-proj/x-cmd/internal/appdir"
)

const (
	markerStart = "# x-cmd completion start"
	markerEnd   = "# x-cmd completion end"
)

var supportedShells = []string{"bash", "zsh", "fish", "powershell"}

func Candidates(args []string) []string {
	prefix := ""
	context := args
	if len(args) > 0 {
		prefix = args[len(args)-1]
		context = args[:len(args)-1]
	}
	options := optionsFor(context)
	result := make([]string, 0, len(options))
	for _, option := range options {
		if strings.HasPrefix(option, prefix) {
			result = append(result, option)
		}
	}
	sort.Strings(result)
	return result
}

func optionsFor(args []string) []string {
	if len(args) == 0 {
		return []string{"completion", "config", "core", "github-mirror", "help", "node", "proxy", "sub", "subscription", "system", "uninstall", "update", "version"}
	}
	if len(args) == 1 {
		switch args[0] {
		case "system":
			return []string{"start", "status", "stop"}
		case "proxy":
			return []string{"disable", "enable", "status"}
		case "update":
			return []string{"check", "install"}
		case "uninstall":
			return []string{"--yes"}
		case "core":
			return []string{"install", "releases", "show"}
		case "config":
			return []string{"set", "show"}
		case "github-mirror":
			return []string{"delete", "set", "show"}
		case "sub", "subscription":
			return []string{"add", "delete", "edit", "list", "nodes", "update"}
		case "node":
			return []string{"add", "delete", "list", "test", "use"}
		case "completion":
			return []string{"install", "uninstall"}
		}
	}
	if len(args) == 2 {
		switch args[0] + " " + args[1] {
		case "completion install", "completion uninstall":
			return supportedShells
		case "core install":
			return []string{"--dir", "--version"}
		case "config set":
			return []string{"--download-url", "--github-mirror", "--listen-port", "--test-url", "--xray-path"}
		case "sub add":
			return []string{"--name", "--url"}
		case "sub edit", "subscription edit":
			return []string{"--name", "--url"}
		case "node add":
			return []string{"--name", "--uri"}
		case "node list":
			return []string{"--subscription"}
		case "node test":
			return []string{"--delete-invalid", "--subscription", "--timeout"}
		}
	}
	return nil
}

func Install(shell string) ([]string, error) {
	shell, err := normalizeShell(shell)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root, err := appdir.Default()
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(root, "completions")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}

	var paths []string
	switch shell {
	case "bash", "zsh":
		scriptPath := filepath.Join(directory, "x-cmd."+shell)
		if err := os.WriteFile(scriptPath, []byte(scriptFor(shell)), 0o600); err != nil {
			return nil, err
		}
		profile := filepath.Join(home, "."+shell+"rc")
		line := fmt.Sprintf("[ -f %s ] && . %s", shellQuote(scriptPath), shellQuote(scriptPath))
		if err := updateProfile(profile, line, true); err != nil {
			return nil, err
		}
		paths = []string{scriptPath, profile}
	case "fish":
		path := filepath.Join(home, ".config", "fish", "completions", "x-cmd.fish")
		if err := writeFile(path, scriptFor(shell)); err != nil {
			return nil, err
		}
		paths = []string{path}
	case "powershell":
		scriptPath := filepath.Join(directory, "x-cmd.ps1")
		if err := os.WriteFile(scriptPath, []byte(scriptFor(shell)), 0o600); err != nil {
			return nil, err
		}
		for _, profile := range powershellProfiles(home) {
			line := fmt.Sprintf("if (Test-Path '%s') { . '%s' }", psQuote(scriptPath), psQuote(scriptPath))
			if err := updateProfile(profile, line, true); err != nil {
				return nil, err
			}
			paths = append(paths, profile)
		}
		paths = append([]string{scriptPath}, paths...)
	}
	return paths, nil
}

func Uninstall(shell string) error {
	if strings.TrimSpace(shell) == "" {
		return UninstallAll()
	}
	shell, err := normalizeShell(shell)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	root, err := appdir.Default()
	if err != nil {
		return err
	}
	var errs []error
	switch shell {
	case "bash", "zsh":
		errs = append(errs, removeFile(filepath.Join(root, "completions", "x-cmd."+shell)))
		errs = append(errs, updateProfile(filepath.Join(home, "."+shell+"rc"), "", false))
	case "fish":
		errs = append(errs, removeFile(filepath.Join(home, ".config", "fish", "completions", "x-cmd.fish")))
	case "powershell":
		errs = append(errs, removeFile(filepath.Join(root, "completions", "x-cmd.ps1")))
		for _, profile := range powershellProfiles(home) {
			errs = append(errs, updateProfile(profile, "", false))
		}
	}
	return errors.Join(errs...)
}

func UninstallAll() error {
	var errs []error
	for _, shell := range supportedShells {
		errs = append(errs, Uninstall(shell))
	}
	return errors.Join(errs...)
}

func normalizeShell(shell string) (string, error) {
	shell = strings.ToLower(strings.TrimSpace(shell))
	if shell == "" {
		if runtime.GOOS == "windows" {
			return "powershell", nil
		}
		shell = filepath.Base(os.Getenv("SHELL"))
	}
	if shell == "pwsh" || shell == "powershell.exe" || shell == "pwsh.exe" {
		shell = "powershell"
	}
	for _, supported := range supportedShells {
		if shell == supported {
			return shell, nil
		}
	}
	return "", fmt.Errorf("不支持的 shell %q，可选: bash、zsh、fish、powershell", shell)
}

func updateProfile(path, line string, install bool) error {
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	content := removeMarkedBlock(string(raw))
	if install {
		content = strings.TrimRight(content, "\r\n")
		if content != "" {
			content += "\n"
		}
		content += markerStart + "\n" + line + "\n" + markerEnd + "\n"
	}
	if errors.Is(err, os.ErrNotExist) && !install {
		return nil
	}
	return writeFile(path, content)
}

func removeMarkedBlock(content string) string {
	for {
		start := strings.Index(content, markerStart)
		if start < 0 {
			return content
		}
		end := strings.Index(content[start:], markerEnd)
		if end < 0 {
			return content
		}
		end = start + end + len(markerEnd)
		if end < len(content) && content[end] == '\r' {
			end++
		}
		if end < len(content) && content[end] == '\n' {
			end++
		}
		content = content[:start] + content[end:]
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func removeFile(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func powershellProfiles(home string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
	}
	return []string{filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func scriptFor(shell string) string {
	switch shell {
	case "bash":
		return `_x_cmd_completion() {
  local current="${COMP_WORDS[COMP_CWORD]}"
  local words=("${COMP_WORDS[@]:1:$COMP_CWORD}")
  COMPREPLY=()
  while IFS= read -r candidate; do COMPREPLY+=("$candidate"); done < <(x-cmd completion candidates "${words[@]}")
}
complete -F _x_cmd_completion x-cmd
`
	case "zsh":
		return `#compdef x-cmd
_x_cmd_completion() {
  local -a candidates
  candidates=("${(@f)$(x-cmd completion candidates "${words[@]:1}")}")
  _describe 'x-cmd' candidates
}
compdef _x_cmd_completion x-cmd
`
	case "fish":
		return `complete -c x-cmd -f -a '(x-cmd completion candidates (commandline -opc)[2..-1] (commandline -ct))'
`
	case "powershell":
		return `Register-ArgumentCompleter -Native -CommandName x-cmd -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $arguments = @($commandAst.CommandElements | Select-Object -Skip 1 | ForEach-Object { $_.Extent.Text })
    x-cmd completion candidates @arguments | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`
	}
	return ""
}
