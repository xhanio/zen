package ulidutil

import (
	"strings"
	"testing"

	"github.com/xhanio/errors"
)

func TestNew_Format(t *testing.T) {
	id := New()
	if len(id) != 26 {
		t.Fatalf("expected 26-char ulid, got %d: %q", len(id), id)
	}
	const allowed = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, c := range id {
		if !strings.ContainsRune(allowed, c) {
			t.Fatalf("invalid char %q in ulid %q", c, id)
		}
	}
}

func TestNew_MonotonicWithinTick(t *testing.T) {
	a := New()
	b := New()
	if a >= b {
		t.Fatalf("expected b > a; got a=%q b=%q", a, b)
	}
}

func TestParse_Valid(t *testing.T) {
	id := New()
	if err := Parse(id); err != nil {
		t.Fatalf("Parse(%q) returned error: %v", id, err)
	}
}

func TestParse_InvalidLength(t *testing.T) {
	err := Parse("too-short")
	if err == nil {
		t.Fatalf("expected error for short id")
	}
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestParse_InvalidCharset(t *testing.T) {
	// '!' is not in the Crockford base32 alphabet.
	err := Parse("01ARZ3NDEKTSV4RRFFQ69G5E!V")
	if err == nil {
		t.Fatalf("expected error for bad char")
	}
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}
