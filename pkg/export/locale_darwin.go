package export

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// systemLocale returns the macOS locale, such as "en_US" or
// "en_IN@rg=uszzzz". Apps started from the Finder have no LANG.
func systemLocale() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
