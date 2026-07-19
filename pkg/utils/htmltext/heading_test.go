package htmltext

import "testing"

func TestStripLeadingHeading(t *testing.T) {
	cases := []struct {
		name    string
		content string
		format  string
		title   string
		want    string
	}{
		{
			name:    "html: strips matching leading h2 inside a wrapper",
			content: `<article><style>.x h2{}</style><h2>变更</h2><p>body</p></article>`,
			format:  "html",
			title:   "变更",
			want:    `<article><style>.x h2{}</style><p>body</p></article>`,
		},
		{
			name:    "html: keeps a leading heading that is not the title",
			content: `<h3>步骤 1</h3><p>body</p>`,
			format:  "html",
			title:   "任务 3",
			want:    `<h3>步骤 1</h3><p>body</p>`,
		},
		{
			name:    "html: only the first heading is removed",
			content: `<h2>T</h2><h2>T</h2>`,
			format:  "html",
			title:   "T",
			want:    `<h2>T</h2>`,
		},
		{
			name:    "html: heading with inline markup, whitespace-tolerant match",
			content: `<h2> <code>build</code> </h2><p>b</p>`,
			format:  "html",
			title:   "build",
			want:    `<p>b</p>`,
		},
		{
			name:    "markdown: strips matching leading heading + trailing blank",
			content: "## 变更\n\nbody here",
			format:  "markdown",
			title:   "变更",
			want:    "body here",
		},
		{
			name:    "markdown: first non-blank line is not a heading",
			content: "some text\n## later",
			format:  "markdown",
			title:   "变更",
			want:    "some text\n## later",
		},
		{
			name:    "text: passes through unchanged",
			content: "# not a heading in text",
			format:  "text",
			title:   "whatever",
			want:    "# not a heading in text",
		},
		{
			name:    "already clean html is a no-op (idempotent)",
			content: `<p>body</p>`,
			format:  "html",
			title:   "变更",
			want:    `<p>body</p>`,
		},
		{
			name:    "empty title is a no-op",
			content: `<h2>x</h2>`,
			format:  "html",
			title:   "  ",
			want:    `<h2>x</h2>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripLeadingHeading(tc.content, tc.format, tc.title)
			if got != tc.want {
				t.Errorf("StripLeadingHeading()\n  got:  %q\n  want: %q", got, tc.want)
			}
		})
	}
}
