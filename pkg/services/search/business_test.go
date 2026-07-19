package search_test

import (
	"context"
	"testing"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/search"
)

func TestSearch_EmptyQuery(t *testing.T) {
	svc := search.New(repository.New(testutil.NewDB(t)))
	_, _, err := svc.Search(context.Background(), "  ", "all", 10)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for empty query, got: %v", err)
	}
}

func TestSearch_InvalidScope(t *testing.T) {
	svc := search.New(repository.New(testutil.NewDB(t)))
	_, _, err := svc.Search(context.Background(), "foo", "bogus", 10)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for bad scope, got: %v", err)
	}
}

func TestSearch_LimitTooLarge(t *testing.T) {
	svc := search.New(repository.New(testutil.NewDB(t)))
	_, _, err := svc.Search(context.Background(), "foo", "all", 200)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for limit>100, got: %v", err)
	}
}

func TestSearch_LimitDefault(t *testing.T) {
	svc := search.New(repository.New(testutil.NewDB(t)))
	_, _, err := svc.Search(context.Background(), "", "all", 0)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest (empty query), got: %v", err)
	}
}
