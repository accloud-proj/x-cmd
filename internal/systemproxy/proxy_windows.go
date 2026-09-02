//go:build windows

package systemproxy

import (
	"fmt"
	"os/exec"
)

const internetSettings = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func Enable(port int) error {
	address := Address(port)
	server := fmt.Sprintf("http=%s;https=%s;socks=%s", address, address, address)
	if err := run("reg", "add", internetSettings, "/v", "ProxyServer", "/t", "REG_SZ", "/d", server, "/f"); err != nil {
		return err
	}
	return run("reg", "add", internetSettings, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
}

func Disable() error {
	return run("reg", "add", internetSettings, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
}

func run(name string, arguments ...string) error {
	if output, err := exec.Command(name, arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
