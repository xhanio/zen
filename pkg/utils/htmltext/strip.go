// Package htmltext extracts plain text from HTML for use as the FTS5
// indexable projection of an entity's content. The goal is "make HTML
// searchable like markdown": tag names and attribute strings must not
// appear in the index, but inline SVG <text> labels should.
package htmltext

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Strip returns the visible text of raw with block-level elements separated
// by '\n' and inline content concatenated without separator. Script and
// style content is dropped. HTML entities are decoded. Returns "" for empty
// input.
func Strip(raw string) string {
	if raw == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader("<html><body>" + raw + "</body></html>"))
	if err != nil {
		return raw
	}
	var b strings.Builder
	walk(doc, &b)
	return strings.TrimSpace(b.String())
}

func walk(n *html.Node, b *strings.Builder) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode {
		switch n.DataAtom {
		case atom.Script, atom.Style:
			return
		}
	}
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, b)
	}
	if n.Type == html.ElementNode && isBlock(n) {
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
	}
}

func isBlock(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Address, atom.Article, atom.Aside, atom.Blockquote,
		atom.Br, atom.Canvas, atom.Dd, atom.Div, atom.Dl, atom.Dt,
		atom.Fieldset, atom.Figcaption, atom.Figure, atom.Footer,
		atom.Form, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6,
		atom.Header, atom.Hr, atom.Li, atom.Main, atom.Nav, atom.Noscript,
		atom.Ol, atom.P, atom.Pre, atom.Section, atom.Table, atom.Tfoot,
		atom.Ul:
		return true
	}
	// SVG <text>/<tspan> are foreign content; their DataAtom is 0. Match by
	// element name so each label becomes its own index token.
	return n.Data == "text" || n.Data == "tspan"
}
