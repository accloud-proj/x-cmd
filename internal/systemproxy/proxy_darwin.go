//go:build darwin

package systemproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

func Enable(port int) error { return configureDarwin(port, true) }
func Disable() error        { return configureDarwin(0, false) }

func configureDarwin(port int, enabled bool) error {
	output, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return err
	}
	state := "off"
	if enabled {
		state = "on"
	}
	for _, service := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if service == "" || strings.HasPrefix(service, "An asterisk") || strings.HasPrefix(service, "*") {
			continue
		}
		if enabled {
			for _, kind := range []string{"-setwebproxy", "-setsecurewebproxy", "-setsocksfirewallproxy"} {
				if value, err := exec.Command("networksetup", kind, service, "127.0.0.1", fmt.Sprint(port)).CombinedOutput(); err != nil {
					return fmt.Errorf("%s: %s", err, value)
				}
			}
		}
		for _, kind := range []string{"-setwebproxystate", "-setsecurewebproxystate", "-setsocksfirewallproxystate"} {
			if value, err := exec.Command("networksetup", kind, service, state).CombinedOutput(); err != nil {
				return fmt.Errorf("%s: %s", err, value)
			}
		}
	}
	return nil
}
