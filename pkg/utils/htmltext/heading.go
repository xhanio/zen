package htmltext

import (
	"regexp"
	"strings"
)

// The first heading element (html) or ATX heading line (markdown). RE2 has no
// backreferences, so the closing tag matches any h1–h6; well-formed headings
// aren't nested, so the non-greedy body still stops at the matching close.
var (
	htmlHeadingRe = regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)
	mdHeadingRe   = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$`)
)

// StripLeadingHeading removes a card body's leading heading when its text
// matches title. A decomposed section's title lives in the card's title field;
// it must not be duplicated as a heading inside the content. Idempotent: if the
// leading heading is absent or doesn't match the title, content is returned
// unchanged. Only "html" and "markdown" carry headings; other formats pass
// through.
func StripLeadingHeading(content, format, title string) string {
	title = strings.TrimSpace(title)
	if title == "" || strings.TrimSpace(content) == "" {
		return content
	}
	switch format {
	case "html":
		return stripHTMLLeadingHeading(content, title)
	case "markdown":
		return stripMarkdownLeadingHeading(content, title)
	default:
		return content
	}
}

func stripHTMLLeadingHeading(content, title string) string {
	loc := htmlHeadingRe.FindStringSubmatchIndex(content)
	if loc == nil {
		return content
	}
	inner := content[loc[2]:loc[3]] // group 1: the heading's inner HTML
	if !strings.EqualFold(strings.TrimSpace(Strip(inner)), title) {
		return content
	}
	return content[:loc[0]] + content[loc[1]:]
}

func stripMarkdownLeadingHeading(content, title string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // skip leading blank lines
		}
		m := mdHeadingRe.FindStringSubmatch(line)
		if m == nil || !strings.EqualFold(strings.TrimSpace(m[1]), title) {
			return content // first non-blank line isn't the title heading
		}
		rest := lines[i+1:]
		if len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
			rest = rest[1:] // drop one blank line the heading left behind
		}
		return strings.Join(rest, "\n")
	}
	return content
}
