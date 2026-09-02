package uninstaller

import "os"

func Remove(configPath, runtimeDir string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return remove(executable, configPath, runtimeDir)
}

func removeData(configPath, runtimeDir string) error {
	if err := os.RemoveAll(runtimeDir); err != nil {
		return err
	}
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(configPath + ".tmp"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
