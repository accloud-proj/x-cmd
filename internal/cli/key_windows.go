//go:build windows

package cli

import (
	"bufio"
	"os"
	"syscall"
	"unsafe"
)

const (
	enableEchoInput = 0x0004
	enableLineInput = 0x0002
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode = kernel32.NewProc("GetConsoleMode")
	setConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func readKey(input *bufio.Reader) error {
	handle := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	if result, _, _ := getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode))); result == 0 {
		_, err := input.ReadString('\n')
		return err
	}
	if result, _, callErr := setConsoleMode.Call(uintptr(handle), uintptr(mode&^(enableEchoInput|enableLineInput))); result == 0 {
		return callErr
	}
	defer setConsoleMode.Call(uintptr(handle), uintptr(mode))
	_, err := input.ReadByte()
	return err
}
