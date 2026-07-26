// Package carddiff computes the stored diff between two card states.
//
// Line structure comes from diffmatchpatch's line mode; the intra-line spans
// come from a second rune-level pass over each del/add pair. Rune-level is the
// whole point: Chinese prose has no spaces, so a whitespace tokenizer would
// mark an entire paragraph changed for a one-character edit.
package carddiff

import (
	"encoding/json"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/xhanio/zen/pkg/types/entity"
)

// MaxDiffBytes caps the serialized diff. A full rewrite of a 1 MB card can
// produce a payload larger than the card itself; past this the row keeps only
// the counts and the UI falls back to showing both bodies.
const MaxDiffBytes = 262144

type Fields struct {
	Title   string
	Summary string
	Content string
	Format  string
}

// Compute returns the serialized diff, whether it was truncated, and the
// added/removed line counts. A truncated diff serializes to "" but still
// reports accurate counts.
func Compute(before, after Fields) (string, bool, int, int) {
	d := entity.CardDiff{
		Fields: fieldRows(before, after),
		Lines:  bodyLines(before.Content, after.Content),
	}

	added, removed := 0, 0
	for _, ln := range d.Lines {
		switch ln.Op {
		case "add":
			added++
		case "del":
			removed++
		}
	}

	raw, err := json.Marshal(d)
	if err != nil || len(raw) > MaxDiffBytes {
		return "", true, added, removed
	}
	return string(raw), false, added, removed
}

func fieldRows(before, after Fields) []entity.DiffField {
	rows := make([]entity.DiffField, 0, 3)
	for _, f := range []struct {
		key         string
		old, latest string
	}{
		{"title", before.Title, after.Title},
		{"summary", before.Summary, after.Summary},
		{"format", before.Format, after.Format},
	} {
		if f.old == f.latest {
			continue
		}
		rows = append(rows, entity.DiffField{Key: f.key, Spans: spans(f.old, f.latest)})
	}
	return rows
}

// spans runs a rune-level diff and cleans it up semantically, so scattered
// single-character edits merge into readable chunks.
func spans(old, latest string) []entity.DiffSpan {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffCleanupSemantic(dmp.DiffMain(old, latest, false))

	out := make([]entity.DiffSpan, 0, len(diffs))
	for _, d := range diffs {
		out = append(out, entity.DiffSpan{Op: opName(d.Type), Text: d.Text})
	}
	return out
}

func opName(t diffmatchpatch.Operation) string {
	switch t {
	case diffmatchpatch.DiffInsert:
		return "add"
	case diffmatchpatch.DiffDelete:
		return "del"
	default:
		return "eq"
	}
}

// terminate ensures a non-empty body ends with a newline. diffmatchpatch's
// line mode keeps the trailing newline as part of each line, so "…\nc" and
// "…\nc\n" compare as different final lines — appending a paragraph would
// otherwise repaint the previous last line as rewritten. Empty stays empty; a
// lone "\n" would invent a phantom line.
func terminate(body string) string {
	if body == "" || strings.HasSuffix(body, "\n") {
		return body
	}
	return body + "\n"
}

// bodyLines diffs the two bodies line by line, then fills in intra-line spans
// for each adjacent del/add pair.
func bodyLines(oldBody, newBody string) []entity.DiffLine {
	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToRunes(terminate(oldBody), terminate(newBody))
	diffs := dmp.DiffCharsToLines(dmp.DiffMain(string(a), string(b), false), lines)

	out := make([]entity.DiffLine, 0)
	for _, d := range diffs {
		op := lineOp(opName(d.Type))
		for _, text := range splitLines(d.Text) {
			out = append(out, entity.DiffLine{Op: op, Text: text})
		}
	}
	pairSpans(out)
	return out
}

func lineOp(op string) string {
	if op == "eq" {
		return "ctx"
	}
	return op
}

// splitLines drops the trailing empty element produced by a text block that
// ends in a newline, so a 3-line block yields exactly 3 lines.
func splitLines(text string) []string {
	parts := strings.Split(text, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// pairSpans walks runs of del lines immediately followed by add lines and
// gives each positional pair its intra-line spans. Unpaired lines (a pure
// insertion or deletion) keep Spans nil — the whole line is the change.
func pairSpans(lines []entity.DiffLine) {
	i := 0
	for i < len(lines) {
		if lines[i].Op != "del" {
			i++
			continue
		}
		delStart := i
		for i < len(lines) && lines[i].Op == "del" {
			i++
		}
		addStart := i
		for i < len(lines) && lines[i].Op == "add" {
			i++
		}
		delCount, addCount := addStart-delStart, i-addStart
		for k := 0; k < delCount && k < addCount; k++ {
			s := spans(lines[delStart+k].Text, lines[addStart+k].Text)
			lines[delStart+k].Spans = s
			lines[addStart+k].Spans = s
		}
	}
}
