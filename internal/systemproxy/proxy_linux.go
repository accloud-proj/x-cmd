//go:build linux

package systemproxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Enable(port int) error {
	if !gnomeProxyAvailable() {
		return writeEnvironmentProxy(port)
	}
	commands := [][]string{
		{"org.gnome.system.proxy", "mode", "manual"},
		{"org.gnome.system.proxy.http", "host", "127.0.0.1"}, {"org.gnome.system.proxy.http", "port", fmt.Sprint(port)},
		{"org.gnome.system.proxy.https", "host", "127.0.0.1"}, {"org.gnome.system.proxy.https", "port", fmt.Sprint(port)},
		{"org.gnome.system.proxy.socks", "host", "127.0.0.1"}, {"org.gnome.system.proxy.socks", "port", fmt.Sprint(port)},
	}
	for _, arguments := range commands {
		if err := gsettings(arguments...); err != nil {
			return err
		}
	}
	return nil
}

func Disable() error {
	if gnomeProxyAvailable() {
		return gsettings("org.gnome.system.proxy", "mode", "none")
	}
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
