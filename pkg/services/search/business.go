package search

import (
	"context"
	"strings"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
)

const (
	scopeAll      = "all"
	scopeCards    = "cards"
	scopeMessages = "messages"

	defaultLimit = 20
	maxLimit     = 100
)

func (m *manager) Search(ctx context.Context, query, scope string, limit int) (cards, msgs []*entity.SearchHit, err error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, errors.BadRequest.Newf("search query is required")
	}
	if scope == "" {
		scope = scopeAll
	}
	switch scope {
	case scopeAll, scopeCards, scopeMessages:
		// ok
	default:
		return nil, nil, errors.BadRequest.Newf("scope must be one of all, cards, messages")
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		return nil, nil, errors.BadRequest.Newf("limit must be %d or less", maxLimit)
	}

	cards = []*entity.SearchHit{}
	msgs = []*entity.SearchHit{}

	if scope == scopeAll || scope == scopeCards {
		cards, err = m.repo.SearchCards(ctx, query, limit)
		if err != nil {
			return nil, nil, errors.Wrap(err)
		}
	}
	if scope == scopeAll || scope == scopeMessages {
		msgs, err = m.repo.SearchMessages(ctx, query, limit)
		if err != nil {
			return nil, nil, errors.Wrap(err)
		}
	}
	return cards, msgs, nil
}
