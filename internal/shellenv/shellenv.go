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
		args = []string{"-NoExit"}
	} else if shell == "cmd" {
		args = []string{"/K"}
	} else if shell == "sh" {
		args = nil
	}
	return shell, args, nil
}

func Run(name string, args, environment []string) error {
	command := exec.Command(name, args...)
	command.Env = environment
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
