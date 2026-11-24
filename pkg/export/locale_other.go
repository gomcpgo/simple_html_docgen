//go:build !darwin && !windows

package export

// systemLocale has nothing beyond the POSIX locale variables to go on.
func systemLocale() string { return "" }
