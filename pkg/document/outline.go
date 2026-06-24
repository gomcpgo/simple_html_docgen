package document

import (
	"html"
	"regexp"
	"strings"
)

// Heading is a single heading parsed from a document's HTML.
type Heading struct {
	Level int    `json:"level"` // 1-6, from <h1>..<h6>
	Text  string `json:"text"`  // visible text, inner tags stripped
}

var (
	headingRe    = regexp.MustCompile(`(?is)<h([1-6])\b[^>]*>(.*?)</h[1-6]>`)
	innerTagRe   = regexp.MustCompile(`(?s)<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

// ParseHeadings extracts the <h1>..<h6> headings from HTML in document order,
// stripping inner tags and attributes so only level and visible text remain.
func ParseHeadings(htmlContent string) []Heading {
	matches := headingRe.FindAllStringSubmatch(htmlContent, -1)
	headings := make([]Heading, 0, len(matches))
	for _, m := range matches {
		level := int(m[1][0] - '0')
		text := innerTagRe.ReplaceAllString(m[2], "")
		text = html.UnescapeString(text)
		text = whitespaceRe.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
		headings = append(headings, Heading{Level: level, Text: text})
	}
	return headings
}
