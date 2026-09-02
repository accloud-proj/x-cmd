//go:build linux

package systemproxy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	proxyMarkerStart = "# >>> x-cmd proxy >>>"
	proxyMarkerEnd   = "# <<< x-cmd proxy <<<"
)

func Enable(port int) error {
	if err := writeShellProxy(port); err != nil {
		return err
	}
	if !gnomeProxyAvailable() {
		if err := writeEnvironmentProxy(port); err != nil {
			_ = removeShellProxy()
			return err
		}
		return nil
	}
	commands := [][]string{
		{"org.gnome.system.proxy", "mode", "manual"},
		{"org.gnome.system.proxy.http", "host", "127.0.0.1"}, {"org.gnome.system.proxy.http", "port", fmt.Sprint(port)},
		{"org.gnome.system.proxy.https", "host", "127.0.0.1"}, {"org.gnome.system.proxy.https", "port", fmt.Sprint(port)},
		{"org.gnome.system.proxy.socks", "host", "127.0.0.1"}, {"org.gnome.system.proxy.socks", "port", fmt.Sprint(port)},
	}
	for _, arguments := range commands {
		if err := gsettings(arguments...); err != nil {
			_ = removeShellProxy()
			return err
		}
	}
	return nil
}

func Disable() error {
	var errs []error
	if gnomeProxyAvailable() {
		errs = append(errs, gsettings("org.gnome.system.proxy", "mode", "none"))
	}
	errs = append(errs, removeShellProxy(), removeEnvironmentProxy())
	return errors.Join(errs...)
}

func removeEnvironmentProxy() error {
	path, err := environmentProxyPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeShellProxy(port int) error {
	profile, shell, err := shellProxyProfile()
	if err != nil || profile == "" {
		return err
	}
	address := Address(port)
	var lines string
	switch shell {
	case "fish":
		lines = fmt.Sprintf("set -gx HTTP_PROXY http://%s\nset -gx HTTPS_PROXY http://%s\nset -gx ALL_PROXY socks5://%s\nset -gx NO_PROXY localhost,127.0.0.1,::1", address, address, address)
	case "powershell":
		lines = fmt.Sprintf("$env:HTTP_PROXY = 'http://%s'\n$env:HTTPS_PROXY = 'http://%s'\n$env:ALL_PROXY = 'socks5://%s'\n$env:NO_PROXY = 'localhost,127.0.0.1,::1'", address, address, address)
	default:
		lines = fmt.Sprintf("export HTTP_PROXY=http://%s\nexport HTTPS_PROXY=http://%s\nexport ALL_PROXY=socks5://%s\nexport NO_PROXY=localhost,127.0.0.1,::1\nexport http_proxy=\"$HTTP_PROXY\"\nexport https_proxy=\"$HTTPS_PROXY\"\nexport all_proxy=\"$ALL_PROXY\"\nexport no_proxy=\"$NO_PROXY\"", address, address, address)
	}
	return updateProxyProfile(profile, lines)
}

func removeShellProxy() error {
	profile, _, err := shellProxyProfile()
	if err != nil || profile == "" {
		return err
	}
	return updateProxyProfile(profile, "")
}

func shellProxyProfile() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	shell := strings.ToLower(filepath.Base(os.Getenv("SHELL")))
	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), shell, nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), shell, nil
	case "sh", "dash", "ash", "ksh":
		return filepath.Join(home, ".profile"), shell, nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "conf.d", "x-cmd-proxy.fish"), shell, nil
	case "pwsh", "powershell":
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), "powershell", nil
	default:
		return "", "", nil
	}
}

func updateProxyProfile(path, lines string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	content := removeProxyBlock(string(raw))
	if lines != "" {
		content = strings.TrimRight(content, "\r\n")
		if content != "" {
			content += "\n"
		}
		content += proxyMarkerStart + "\n" + lines + "\n" + proxyMarkerEnd + "\n"
	}
	if errors.Is(err, os.ErrNotExist) && lines == "" {
		return nil
	}
	if lines == "" && strings.TrimSpace(content) == "" {
		removeErr := os.Remove(path)
		if errors.Is(removeErr, os.ErrNotExist) {
			return nil
		}
		return removeErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func removeProxyBlock(content string) string {
	start := strings.Index(content, proxyMarkerStart)
	if start < 0 {
		return content
	}
	end := strings.Index(content[start:], proxyMarkerEnd)
	if end < 0 {
		return content
	}
	end += start + len(proxyMarkerEnd)
	if end < len(content) && content[end] == '\r' {
		end++
	}
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + content[end:]
}

func gnomeProxyAvailable() bool {
	output, err := exec.Command("gsettings", "list-schemas").Output()
	if err != nil {
		return false
	}
	for _, schema := range strings.Fields(string(output)) {
		if schema == "org.gnome.system.proxy" {
			return true
		}
	}
	return false
}

func writeEnvironmentProxy(port int) error {
	path, err := environmentProxyPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	address := Address(port)
	content := fmt.Sprintf("HTTP_PROXY=http://%s\nHTTPS_PROXY=http://%s\nALL_PROXY=socks5://%s\nNO_PROXY=localhost,127.0.0.1,::1\n", address, address, address)
	return os.WriteFile(path, []byte(content), 0o600)
}

func environmentProxyPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "environment.d", "90-x-cmd-proxy.conf"), nil
}

func gsettings(arguments ...string) error {
	arguments = append([]string{"set"}, arguments...)
	if output, err := exec.Command("gsettings", arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, output)
	}
	return nil
}
