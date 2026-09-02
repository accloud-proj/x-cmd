//go:build windows

package systemproxy

import (
	"fmt"
	"os/exec"
	"syscall"
)

const internetSettings = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func Enable(port int) error {
	address := Address(port)
	server := fmt.Sprintf("http=%s;https=%s;socks=%s", address, address, address)
	if err := run("reg", "add", internetSettings, "/v", "ProxyServer", "/t", "REG_SZ", "/d", server, "/f"); err != nil {
		return err
	}
	if err := run("reg", "add", internetSettings, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f"); err != nil {
		return err
	}
	return notifySettingsChanged()
}

func Disable() error {
	if err := run("reg", "add", internetSettings, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f"); err != nil {
		return err
	}
	return notifySettingsChanged()
}

func notifySettingsChanged() error {
	procedure := syscall.NewLazyDLL("wininet.dll").NewProc("InternetSetOptionW")
	for _, option := range []uintptr{39, 37} {
		result, _, callErr := procedure.Call(0, option, 0, 0)
		if result == 0 {
			return fmt.Errorf("刷新系统代理设置失败: %w", callErr)
		}
	}
	return nil
}

func run(name string, arguments ...string) error {
	if output, err := exec.Command(name, arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
