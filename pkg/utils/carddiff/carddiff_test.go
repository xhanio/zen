package carddiff_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/carddiff"
)

func decode(t *testing.T, raw string) entity.CardDiff {
	t.Helper()
	var d entity.CardDiff
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("unmarshal diff: %v", err)
	}
	return d
}

func TestCompute_CountsAddedAndRemovedLines(t *testing.T) {
	before := carddiff.Fields{Title: "t", Content: "a\nb\nc", Format: "markdown"}
	after := carddiff.Fields{Title: "t", Content: "a\nB\nc\nd", Format: "markdown"}

	raw, truncated, added, removed := carddiff.Compute(before, after)
	if truncated {
		t.Fatalf("unexpected truncation")
	}
	if added != 2 || removed != 1 {
		t.Fatalf("counts = +%d -%d, want +2 -1", added, removed)
	}
	d := decode(t, raw)
	if len(d.Lines) == 0 {
		t.Fatalf("no lines produced")
	}
}

// The load-bearing test: Chinese prose has no spaces, so a whitespace
// tokenizer would mark the entire paragraph changed. Rune-level diffing must
// isolate the changed characters.
func TestCompute_ChineseParagraphHighlightsOnlyChangedRun(t *testing.T) {
	before := carddiff.Fields{Content: "版本行只记对话，先后由时间线决定，这一点很重要。", Format: "markdown"}
	after := carddiff.Fields{Content: "快照行只记对话，先后由时间线决定，这一点很重要。", Format: "markdown"}

	raw, _, _, _ := carddiff.Compute(before, after)
	d := decode(t, raw)

	var changed strings.Builder
	for _, ln := range d.Lines {
		for _, s := range ln.Spans {
			if s.Op != "eq" {
				changed.WriteString(s.Text)
			}
		}
	}
	got := changed.String()
	if got == "" {
		t.Fatalf("no changed spans produced")
	}
	// "版本" deleted + "快照" added = 4 runes of change. Anything approaching
	// the paragraph length means the tokenizer collapsed the whole line.
	if len([]rune(got)) > 8 {
		t.Fatalf("changed span too wide (%q, %d runes) — whole-paragraph repaint",
			got, len([]rune(got)))
	}
}

func TestCompute_FieldRowsForTitleAndFormat(t *testing.T) {
	before := carddiff.Fields{Title: "old title", Content: "same", Format: "markdown"}
	after := carddiff.Fields{Title: "new title", Content: "same", Format: "html"}

	raw, _, _, _ := carddiff.Compute(before, after)
	d := decode(t, raw)

	keys := map[string]bool{}
	for _, f := range d.Fields {
		keys[f.Key] = true
	}
	if !keys["title"] || !keys["format"] {
		t.Fatalf("field rows = %v, want title and format", keys)
	}
	if keys["summary"] {
		t.Fatalf("unchanged summary must not produce a field row")
	}
}

func TestCompute_TruncatesOversizedDiff(t *testing.T) {
	before := carddiff.Fields{Content: strings.Repeat("alpha beta gamma\n", 40000), Format: "markdown"}
	after := carddiff.Fields{Content: strings.Repeat("delta epsilon zeta\n", 40000), Format: "markdown"}

	raw, truncated, added, removed := carddiff.Compute(before, after)
	if !truncated {
		t.Fatalf("expected truncation for an oversized diff")
	}
	if raw != "" {
		t.Fatalf("truncated diff must store an empty payload, got %d bytes", len(raw))
	}
	if added == 0 || removed == 0 {
		t.Fatalf("counts must survive truncation, got +%d -%d", added, removed)
	}
}
