//go:build windows

package uninstaller

import "syscall"

func detachedProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008}
}
