package export

import "testing"

func TestPageSizeForLocale(t *testing.T) {
	tests := []struct {
		env    map[string]string
		system string
		want   string
	}{
		{map[string]string{"LANG": "en_US.UTF-8"}, "", "letter"},
		{map[string]string{"LANG": "en_GB.UTF-8"}, "", "a4"},
		{map[string]string{"LANG": "en_IN"}, "", "a4"},
		{map[string]string{"LANG": "es_MX.UTF-8"}, "", "letter"},
		{map[string]string{"LANG": "fr_CA"}, "", "letter"},
		{map[string]string{"LANG": "de_DE@euro"}, "", "a4"},
		// LC_PAPER is the setting for paper and wins over LANG; LC_ALL wins over both.
		{map[string]string{"LANG": "en_US.UTF-8", "LC_PAPER": "de_DE.UTF-8"}, "", "a4"},
		{map[string]string{"LANG": "de_DE", "LC_PAPER": "en_US", "LC_ALL": "en_GB"}, "", "a4"},
		// A locale with no region says nothing about paper; ask the system.
		{map[string]string{"LANG": "C.UTF-8"}, "en_US", "letter"},
		{map[string]string{"LANG": "POSIX"}, "en_GB", "a4"},
		{map[string]string{"LANG": "en"}, "en_US", "letter"},
		// System locale forms: macOS (with a region override) and Windows.
		{nil, "en_US", "letter"},
		{nil, "en-US", "letter"},
		{nil, "en_IN@rg=uszzzz", "letter"},
		{nil, "en_US@rg=gbzzzz", "a4"},
		{nil, "zh-Hans-CN", "a4"},
		{nil, "es-419", "a4"},
		// Nothing known: A4, the paper most of the world uses.
		{nil, "", "a4"},
	}
	for _, tt := range tests {
		getenv := func(k string) string { return tt.env[k] }
		system := func() string { return tt.system }
		if got := pageSizeForLocale(getenv, system); got != tt.want {
			t.Errorf("env %v, system %q: got %q, want %q", tt.env, tt.system, got, tt.want)
		}
	}
}

func TestParsePageSize(t *testing.T) {
	for _, s := range []string{"a4", "A4", "letter", " Letter ", "legal"} {
		if _, err := parsePageSize(s); err != nil {
			t.Errorf("parsePageSize(%q): %v", s, err)
		}
	}
	for _, s := range []string{"tabloid", "a5", "8.5x11"} {
		if _, err := parsePageSize(s); err == nil {
			t.Errorf("parsePageSize(%q) should fail", s)
		}
	}
}
