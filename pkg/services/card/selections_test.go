package card_test

import (
	"context"
	"testing"
)

func TestSelections_RejectsMalformedCardID(t *testing.T) {
	svc, _, _ := newCardCtx(t)
	if _, err := svc.Selections(context.Background(), "not-a-ulid"); err == nil {
		t.Fatal("want an error for a malformed card id, got nil")
	}
}

func TestSelections_EmptyForUnknownCard(t *testing.T) {
	svc, _, _ := newCardCtx(t)
	got, err := svc.Selections(context.Background(), "01KYGAKSG0RD8AZGEFNKB1N7VB")
	if err != nil {
		t.Fatalf("want no error for an unknown card, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 selections, got %d", len(got))
	}
}
