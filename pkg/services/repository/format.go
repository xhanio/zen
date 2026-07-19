package repository

import "github.com/xhanio/zen/pkg/utils/htmltext"

// normalizeFormat fills in the default ("markdown") when entity callers
// leave Format empty. The DB CHECK constraint rejects "", so this guard
// keeps older callers and freshly-zero-valued entities valid.
func normalizeFormat(format string) string {
	if format == "" {
		return "markdown"
	}
	return format
}

// searchHintFor returns the FTS5-indexable plain-text projection of a
// content blob given its format. HTML content has its tags stripped so
// search terms match visible text but not element/attribute names; markdown
// and text formats are indexed verbatim.
func searchHintFor(format, content string) string {
	if normalizeFormat(format) == "html" {
		return htmltext.Strip(content)
	}
	return content
}
