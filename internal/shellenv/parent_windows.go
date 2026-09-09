//go:build windows

package shellenv

import (
	"os"
	"syscall"
	"unsafe"
)

const processQueryLimitedInformation = 0x1000

var (
	parentKernel32        = syscall.NewLazyDLL("kernel32.dll")
	openProcess           = parentKernel32.NewProc("OpenProcess")
	queryFullProcessImage = parentKernel32.NewProc("QueryFullProcessImageNameW")
)

func parentProcessName() string {
	handle, _, _ := openProcess.Call(processQueryLimitedInformation, 0, uintptr(os.Getppid()))
	if handle == 0 {
		return ""
	}
	defer syscall.CloseHandle(syscall.Handle(handle))
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	result, _, _ := queryFullProcessImage.Call(handle, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
	if result == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer[:size])
}
