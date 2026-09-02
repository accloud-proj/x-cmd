//go:build linux

package systemproxy

import (
	"fmt"
	"os/exec"
)

func Enable(port int) error {
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

func Disable() error { return gsettings("org.gnome.system.proxy", "mode", "none") }

func gsettings(arguments ...string) error {
	arguments = append([]string{"set"}, arguments...)
	if output, err := exec.Command("gsettings", arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, output)
	}
	return nil
}
