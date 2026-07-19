package htmltext

import "testing"

func TestStrip_PlainBlocks(t *testing.T) {
	got := Strip(`<h1>Title</h1><p>Hello <strong>world</strong>.</p>`)
	want := "Title\nHello world."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStrip_ScriptAndStyleDropped(t *testing.T) {
	got := Strip(`<p>visible</p><script>alert('x')</script><style>p{}</style>`)
	if got != "visible" {
		t.Fatalf("got %q want %q", got, "visible")
	}
}

func TestStrip_EntitiesDecoded(t *testing.T) {
	got := Strip(`<p>5 &lt; 10 &amp; 11 &gt; 10</p>`)
	if got != "5 < 10 & 11 > 10" {
		t.Fatalf("got %q", got)
	}
}

func TestStrip_SVGTextIncluded(t *testing.T) {
	got := Strip(`<svg><text x="0" y="0">labelA</text><text>labelB</text></svg>`)
	if got != "labelA\nlabelB" {
		t.Fatalf("got %q", got)
	}
}

func TestStrip_PreservesWhitespaceInsideText(t *testing.T) {
	got := Strip(`<p>multi   spaces</p>`)
	if got != "multi   spaces" {
		t.Fatalf("got %q", got)
	}
}

func TestStrip_EmptyAndPlain(t *testing.T) {
	if Strip("") != "" {
		t.Fatal("empty")
	}
	if Strip("plain text") != "plain text" {
		t.Fatalf("got %q", Strip("plain text"))
	}
}
