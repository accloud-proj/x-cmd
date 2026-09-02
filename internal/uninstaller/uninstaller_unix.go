//go:build !windows

package uninstaller

import "os"

func remove(executable, configPath, runtimeDir string) error {
	if err := removeData(configPath, runtimeDir); err != nil {
		return err
	}
	if err := os.Remove(executable + ".old"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Remove(executable)
}
