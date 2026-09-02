package appdir

import (
	"os"
	"path/filepath"
	"runtime"
)

func Default() (string, error) {
	if runtime.GOOS == "windows" {
		directory, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(directory, "x-cmd"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "x-cmd"), nil
}
