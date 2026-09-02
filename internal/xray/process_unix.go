//go:build !windows

package xray

import (
	"os/exec"
	"syscall"
)

func setDetachedProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
