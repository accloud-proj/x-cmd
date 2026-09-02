//go:build windows

package uninstaller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func remove(executable, configPath, runtimeDir string) error {
	if err := removeData(configPath, runtimeDir); err != nil {
		return err
	}
	renamed := executable + ".uninstalling"
	_ = os.Remove(renamed)
	if err := os.Rename(executable, renamed); err != nil {
		return fmt.Errorf("准备删除程序失败: %w", err)
	}
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("x-cmd-uninstall-%d.cmd", os.Getpid()))
	script := fmt.Sprintf("@echo off\r\nping 127.0.0.1 -n 2 >NUL\r\ndel /F /Q \"%s\"\r\ndel /F /Q \"%s\"\r\ndel /F /Q \"%%~f0\"\r\n", renamed, executable+".old")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		_ = os.Rename(renamed, executable)
		return err
	}
	command := exec.Command("cmd.exe", "/C", scriptPath)
	command.SysProcAttr = detachedProcessAttributes()
	if err := command.Start(); err != nil {
		_ = os.Remove(scriptPath)
		_ = os.Rename(renamed, executable)
		return fmt.Errorf("启动卸载清理程序失败: %w", err)
	}
	return command.Process.Release()
}
