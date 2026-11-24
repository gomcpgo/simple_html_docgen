package export

import (
	"syscall"
	"unsafe"
)

var getUserDefaultLocaleName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")

// systemLocale returns the Windows user locale, such as "en-US".
func systemLocale() string {
	const localeNameMaxLength = 85
	var buf [localeNameMaxLength]uint16
	n, _, _ := getUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), localeNameMaxLength)
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
