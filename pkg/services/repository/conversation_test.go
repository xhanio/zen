package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func TestCreateAndGetConversation(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	now := time.Now()
	c := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         "test chat",
		AnchorKind:    nil,
		AnchorID:      nil,
		CreatedAt:     now,
		LastMessageAt: now,
	}
	if err := r.CreateConversation(ctx, c); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	got, err := r.GetConversation(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.Title != "test chat" {
		t.Fatalf("title mismatch: got %q", got.Title)
	}
	if got.AnchorKind != nil {
		t.Fatalf("AnchorKind should be nil for standalone, got %v", *got.AnchorKind)
	}
}

func TestCreateConversation_WithAnchor(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	kind := "card"
	id := ulidutil.New()
	c := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         "anchored",
		AnchorKind:    &kind,
		AnchorID:      &id,
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	if err := r.CreateConversation(ctx, c); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
}

func TestCreateConversation_RejectsHalfAnchor(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	kind := "card"
	c := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         "half-anchored",
		AnchorKind:    &kind,
		AnchorID:      nil,
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	if err := r.CreateConversation(ctx, c); err == nil {
		t.Fatalf("expected DB error for half-anchored conversation, got nil")
	}
}

func TestListConversationsByAnchor(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	cardKind := "card"
	cardID := ulidutil.New()
	groupKind := "group"
	groupID := ulidutil.New()
	for i, anchor := range []struct{ k, id string }{
		{cardKind, cardID},
		{cardKind, cardID},
		{groupKind, groupID},
	} {
		k, id := anchor.k, anchor.id
		c := &entity.Conversation{
			ID:            ulidutil.New(),
			Title:         "x",
			AnchorKind:    &k,
			AnchorID:      &id,
			CreatedAt:     time.Now().Add(time.Duration(i) * time.Second),
			LastMessageAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := r.CreateConversation(ctx, c); err != nil {
			t.Fatalf("CreateConversation %d: %v", i, err)
		}
	}
	got, err := r.ListConversationsByAnchor(ctx, "card", cardID)
	if err != nil {
		t.Fatalf("ListConversationsByAnchor: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 conversations on card, got %d", len(got))
	}
}
