package zenbackend

import (
	"strings"
	"testing"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/types/api"
)

func TestCategorize_NotFound(t *testing.T) {
	eb := &api.ErrorBody{Status: 404, Kind: "NotFound", Source: "zen-backend", Message: "group not found"}
	err := categorize(eb)
	if !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "group not found") {
		t.Fatalf("expected message preserved, got: %v", err)
	}
}

func TestCategorize_Conflict(t *testing.T) {
	eb := &api.ErrorBody{Status: 409, Kind: "Conflict", Source: "zen-backend", Message: "name taken"}
	if !errors.Is(categorize(eb), errors.Conflict) {
		t.Fatal("expected Conflict")
	}
}

func TestCategorize_BadRequest(t *testing.T) {
	eb := &api.ErrorBody{Status: 400, Kind: "BadRequest", Source: "zen-backend", Message: "invalid"}
	if !errors.Is(categorize(eb), errors.BadRequest) {
		t.Fatal("expected BadRequest")
	}
}

func TestCategorize_Internal(t *testing.T) {
	eb := &api.ErrorBody{Status: 500, Kind: "Internal", Source: "zen-backend", Message: "oops"}
	if !errors.Is(categorize(eb), errors.Internal) {
		t.Fatal("expected Internal")
	}
}

func TestCategorize_Unavailable(t *testing.T) {
	eb := &api.ErrorBody{Status: 503, Kind: "Unavailable", Source: "zen-backend", Message: "db"}
	if !errors.Is(categorize(eb), errors.Unavailable) {
		t.Fatal("expected Unavailable")
	}
}

func TestCategorize_UnknownKind_PassesThrough(t *testing.T) {
	// An ErrorBody with a Kind that isn't a registered category should NOT
	// be silently re-categorized. Return the raw ErrorBody so the caller
	// gets full structured access.
	eb := &api.ErrorBody{Status: 418, Kind: "Teapot", Source: "zen-backend", Message: "short and stout"}
	err := categorize(eb)
	if errors.Is(err, errors.BadRequest) || errors.Is(err, errors.Internal) {
		t.Fatalf("unknown Kind must not be fabricated into a category, got: %v", err)
	}
	// Caller can still type-assert to access the structured fields.
	gotEB, ok := err.(*api.ErrorBody)
	if !ok || gotEB.Status != 418 {
		t.Fatalf("expected raw *api.ErrorBody passthrough, got: %T %v", err, err)
	}
}

func TestCategorize_NonErrorBody_Wraps(t *testing.T) {
	// Transport-level errors (DNS failure, connection refused, etc.) come
	// through as plain errors, not *api.ErrorBody. They should be wrapped
	// without an invented category.
	plain := errors.Newf("connection refused")
	err := categorize(plain)
	if errors.Is(err, errors.NotFound) {
		t.Fatal("transport error must not get a fabricated category")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected wrapped message, got: %v", err)
	}
}
