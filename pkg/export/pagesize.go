package export

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gomcpgo/htmltodocx"
)

// PageSizes lists the page_size values export_document accepts.
var PageSizes = []string{"a4", "letter", "legal"}

// pageSize is a paper size for DOCX and PDF export.
type pageSize struct {
	name string
	docx htmltodocx.PageSize
	// Width and height in inches, for Chrome's PDF printer.
	widthIn, heightIn float64
}

var pageSizes = map[string]pageSize{
	"a4":     {"a4", htmltodocx.A4, 210 / 25.4, 297 / 25.4},
	"letter": {"letter", htmltodocx.Letter, 8.5, 11},
	"legal":  {"legal", htmltodocx.Legal, 8.5, 14},
}

func parsePageSize(s string) (pageSize, error) {
	ps, ok := pageSizes[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return pageSize{}, fmt.Errorf("unsupported page size %q (must be one of %s)", s, strings.Join(PageSizes, ", "))
	}
	return ps, nil
}

// letterRegions are the regions that use US Letter paper, from the Unicode
// CLDR paper size data. Everywhere else uses A4.
var letterRegions = map[string]bool{
	"BZ": true, "CA": true, "CL": true, "CO": true, "CR": true, "GT": true, "MX": true,
	"NI": true, "PA": true, "PH": true, "PR": true, "SV": true, "US": true, "VE": true,
}

var (
	systemPageSizeOnce sync.Once
	systemPageSize     string
)

// DefaultPageSize returns the page size for the user's locale: "letter" in
// regions that use US Letter, "a4" elsewhere. It is worked out once.
func DefaultPageSize() string {
	systemPageSizeOnce.Do(func() {
		systemPageSize = pageSizeForLocale(os.Getenv, systemLocale)
	})
	return systemPageSize
}

// pageSizeForLocale picks the page size from the POSIX locale variables, in
// their order of precedence, and asks the system when the locale they name
// has no region (C, POSIX, or unset, as for apps started from the macOS
// Finder or on Windows).
func pageSizeForLocale(getenv func(string) string, system func() string) string {
	region := ""
	for _, k := range []string{"LC_ALL", "LC_PAPER", "LANG"} {
		if v := getenv(k); v != "" {
			region = regionOf(v)
			break
		}
	}
	if region == "" {
		region = regionOf(system())
	}
	if letterRegions[region] {
		return "letter"
	}
	return "a4"
}

// regionOf returns the upper-case region of a locale name such as
// "en_US.UTF-8", "en-US", "zh-Hans-CN" or "en_IN@rg=uszzzz" (a macOS locale
// whose region format was changed to the US), or "" if it names none.
func regionOf(locale string) string {
	locale, mods, _ := strings.Cut(locale, "@")
	for _, m := range strings.Split(mods, ";") {
		if v, ok := strings.CutPrefix(strings.ToLower(m), "rg="); ok && len(v) >= 2 {
			return strings.ToUpper(v[:2])
		}
	}
	locale, _, _ = strings.Cut(locale, ".")
	parts := strings.FieldsFunc(locale, func(r rune) bool { return r == '_' || r == '-' })
	for _, p := range parts[min(1, len(parts)):] {
		if len(p) == 2 && isLetters(p) || len(p) == 3 && isDigits(p) {
			return strings.ToUpper(p)
		}
	}
	return ""
}

func isLetters(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
