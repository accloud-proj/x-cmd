package shellenv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var proxyKeys = map[string]bool{
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
}

func Detect() (string, error) {
	return Normalize("")
}

func Normalize(shell string) (string, error) {
	shell = shellName(shell)
	if shell == "" {
		shell = detect(runtime.GOOS, os.Getenv, parentProcessName())
	}
	if shell == "pwsh" {
		return shell, nil
	}
	switch shell {
	case "sh", "bash", "zsh", "fish", "powershell", "cmd":
		return shell, nil
	default:
		return "", fmt.Errorf("不支持的 shell %q，可选: sh, bash, zsh, fish, powershell, pwsh, cmd", shell)
	}
}

func detect(goos string, getenv func(string) string, parent string) string {
	if goos != "windows" {
		shell := shellName(getenv("SHELL"))
		if shell == "" {
			return "sh"
		}
		return shell
	}
	parent = shellName(parent)
	switch parent {
	case "pwsh", "powershell", "cmd":
		return parent
	}
	modulePath := strings.ToLower(getenv("PSModulePath"))
	if getenv("POWERSHELL_DISTRIBUTION_CHANNEL") != "" || strings.Contains(modulePath, `\powershell\7\`) || strings.Contains(modulePath, `/powershell/7/`) {
		return "pwsh"
	}
	commandShell := shellName(getenv("COMSPEC"))
	if commandShell != "" {
		return commandShell
	}
	return "powershell"
}

func shellName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, `\`, "/")
	base := filepath.Base(value)
	return strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
}

func Activation(shell string, port int) (string, error) {
	shell, err := Normalize(shell)
	if err != nil {
		return "", err
	}
	values := Values(port)
	var lines []string
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"} {
		value := values[key]
		switch shell {
		case "fish":
			lines = append(lines, fmt.Sprintf("set -gx %s %s; set -gx %s %s", key, value, strings.ToLower(key), value))
		case "powershell", "pwsh":
			lines = append(lines, fmt.Sprintf("$env:%s = '%s'; $env:%s = '%s'", key, value, strings.ToLower(key), value))
		case "cmd":
			lines = append(lines, fmt.Sprintf("set %s=%s", key, value), fmt.Sprintf("set %s=%s", strings.ToLower(key), value))
		default:
			lines = append(lines, fmt.Sprintf("export %s=%s; export %s=%s", key, value, strings.ToLower(key), value))
		}
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func Environment(current []string, port int) []string {
	result := make([]string, 0, len(current)+8)
	for _, item := range current {
		key, _, _ := strings.Cut(item, "=")
		if !proxyKeys[strings.ToUpper(key)] {
			result = append(result, item)
		}
	}
	values := Values(port)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"} {
		result = append(result, key+"="+values[key], strings.ToLower(key)+"="+values[key])
	}
	return result
}

func EnvironmentForShell(current []string, port int, shell string) []string {
	environment := Environment(current, port)
	switch shell {
	case "sh":
		return setEnvironmentValue(environment, "PS1", "\033[1;36m[proxy]\033[0m $ ")
	case "cmd":
		return setEnvironmentValue(environment, "PROMPT", "[proxy] $P$G ")
	default:
		return environment
	}
}

func Values(port int) map[string]string {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	return map[string]string{
		"HTTP_PROXY": "http://" + address, "HTTPS_PROXY": "http://" + address,
		"ALL_PROXY": "socks5://" + address, "NO_PROXY": "localhost,127.0.0.1,::1",
	}
}

func Command(shell string) (string, []string, error) {
	shell, err := Normalize(shell)
	if err != nil {
		return "", nil, err
	}
	args := []string{"-i"}
	if shell == "powershell" || shell == "pwsh" {
		args = []string{"-NoExit", "-Command", "$global:xCmdOriginalPrompt = (Get-Command prompt).ScriptBlock; function global:prompt { Write-Host -NoNewline -ForegroundColor Cyan '[proxy] '; & $global:xCmdOriginalPrompt }"}
	} else if shell == "cmd" {
		args = []string{"/K"}
	} else if shell == "sh" {
		args = nil
	} else if shell == "fish" {
		args = []string{"-i", "-C", "function fish_prompt; set_color cyan; printf '[proxy] '; set_color normal; printf '%s> ' (prompt_pwd); end"}
	}
	return shell, args, nil
}

func preparePrompt(name string, args, environment []string) ([]string, []string, func(), error) {
	switch filepath.Base(name) {
	case "bash":
		home := environmentValue(environment, "HOME")
		rcFile := filepath.Join(home, ".bashrc")
		if home == "" {
			rcFile = ""
		}
		directory, err := os.MkdirTemp("", "x-cmd-shell-")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("创建 Shell 启动配置失败: %w", err)
		}
		rcPath := filepath.Join(directory, "bashrc")
		rc := ""
		if rcFile != "" {
			rc += "if [ -r " + shellQuote(rcFile) + " ]; then . " + shellQuote(rcFile) + "; fi\n"
		}
		rc += `__x_cmd_proxy_prompt() {
  case "$PS1" in *"[proxy]"*) return ;; esac
  case "$PS1" in
    $'\n'*) PS1=$'\n\\[\\e[1;36m\\][proxy]\\[\\e[0m\\]'${PS1#$'\n'} ;;
    *) PS1='\[\e[1;36m\][proxy]\[\e[0m\] '"$PS1" ;;
  esac
}
case "$(declare -p PROMPT_COMMAND 2>/dev/null)" in
  "declare -a"*) PROMPT_COMMAND+=(__x_cmd_proxy_prompt) ;;
  *) PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND; }__x_cmd_proxy_prompt" ;;
esac
__x_cmd_proxy_prompt
`
		if err := os.WriteFile(rcPath, []byte(rc), 0o600); err != nil {
			_ = os.RemoveAll(directory)
			return nil, nil, nil, fmt.Errorf("写入 Bash 启动配置失败: %w", err)
		}
		return append([]string{"--rcfile", rcPath}, args...), environment, func() { _ = os.RemoveAll(directory) }, nil
	case "zsh":
		originalZdotdir := environmentValue(environment, "ZDOTDIR")
		if originalZdotdir == "" {
			originalZdotdir = environmentValue(environment, "HOME")
		}
		if originalZdotdir == "" {
			originalZdotdir = "."
		}
		directory, err := os.MkdirTemp("", "x-cmd-shell-")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("创建 Shell 启动配置失败: %w", err)
		}
		zshenv := "ZDOTDIR=" + shellQuote(originalZdotdir) + "\n"
		zshenv += "if [ -r " + shellQuote(filepath.Join(originalZdotdir, ".zshenv")) + " ]; then source " + shellQuote(filepath.Join(originalZdotdir, ".zshenv")) + "; fi\n"
		zshenv += "ZDOTDIR=" + shellQuote(directory) + "\n"
		zshrc := "ZDOTDIR=" + shellQuote(originalZdotdir) + "\n"
		zshrc += "if [ -r " + shellQuote(filepath.Join(originalZdotdir, ".zshrc")) + " ]; then source " + shellQuote(filepath.Join(originalZdotdir, ".zshrc")) + "; fi\n"
		zshrc += "ZDOTDIR=" + shellQuote(directory) + "\n"
		zshrc += `__x_cmd_original_prompt=$PROMPT
__x_cmd_render_proxy_prompt() {
  local exit_status=$?
  local prompt_expression=${__x_cmd_original_prompt//\$\?/$exit_status}
  local rendered
  eval "rendered=$prompt_expression" || return
  while [[ "$rendered" == $'\n'* ]]; do
    rendered=${rendered#$'\n'}
  done
  print -rn -- "%F{cyan}[proxy]%f $rendered"
}
PROMPT='$( __x_cmd_render_proxy_prompt )'
typeset +x ZDOTDIR
`
		for file, content := range map[string]string{".zshenv": zshenv, ".zshrc": zshrc} {
			if err := os.WriteFile(filepath.Join(directory, file), []byte(content), 0o600); err != nil {
				_ = os.RemoveAll(directory)
				return nil, nil, nil, fmt.Errorf("写入 Zsh 启动配置失败: %w", err)
			}
		}
		environment = setEnvironmentValue(environment, "ZDOTDIR", directory)
		return args, environment, func() { _ = os.RemoveAll(directory) }, nil
	case "fish":
		return args, environment, func() {}, nil
	default:
		return args, environment, func() {}, nil
	}
}

func environmentValue(environment []string, key string) string {
	for _, item := range environment {
		name, value, found := strings.Cut(item, "=")
		if found && strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}

func setEnvironmentValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		if !strings.EqualFold(name, key) {
			result = append(result, item)
		}
	}
	return append(result, key+"="+value)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func Run(name string, args, environment []string) error {
	args, environment, cleanup, err := preparePrompt(filepath.Base(name), args, environment)
	if err != nil {
		return err
	}
	defer cleanup()
	command := exec.Command(name, args...)
	command.Env = environment
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
