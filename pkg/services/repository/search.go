package repository

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
)

// FTS5 snippet args: snippet(<table>, <col>, <start>, <end>, <ellipsis>, <maxtokens>)
// Column 1 is search_hint (column 0 is title) per the migration order. It must
// be search_hint, not content: cards_fts is an external-content table, so
// snippet() reconstructs from the cards column of the same name — search_hint
// holds the stripped text that was indexed, content holds raw HTML.
const cardSearchSQL = `
SELECT
    'card' AS kind,
    cards.id AS id,
    cards.title AS title,
    snippet(cards_fts, 1, '<mark>', '</mark>', '…', 32) AS snippet,
    cards.group_id AS group_id
FROM cards
JOIN cards_fts ON cards_fts.rowid = cards.rowid
WHERE cards_fts MATCH ?
-- Weight the title column (col 0) 10x over search_hint (col 1). A title match is
-- a strong relevance signal; without this a card whose term is only in its title
-- — e.g. a decomposed document, whose body is empty — ranks below every card
-- with the term in a long body and falls past the result limit.
ORDER BY bm25(cards_fts, 10.0, 1.0)
LIMIT ?
`

// cardAncestorTitlesSQL walks parent_card_id up from a card, returning its
// ancestor titles root-first (excluding the card itself). Used to render the
// breadcrumb ahead of a search hit's title.
const cardAncestorTitlesSQL = `
WITH RECURSIVE anc(id, parent_card_id, title, depth) AS (
    SELECT id, parent_card_id, title, 0 FROM cards WHERE id = ?
    UNION ALL
    SELECT c.id, c.parent_card_id, c.title, anc.depth + 1
    FROM cards c JOIN anc ON c.id = anc.parent_card_id
)
SELECT title FROM anc WHERE depth > 0 ORDER BY depth DESC
`

const messageSearchSQL = `
SELECT
    'message' AS kind,
    messages.id AS id,
    conversations.title AS title,
    snippet(messages_fts, 1, '<mark>', '</mark>', '…', 32) AS snippet,
    '' AS group_id,
    messages.conversation_id AS conversation_id
FROM messages
JOIN messages_fts ON messages_fts.rowid = messages.rowid
JOIN conversations ON conversations.id = messages.conversation_id
WHERE messages_fts MATCH ?
ORDER BY rank
LIMIT ?
`

// searchRow is the flat shape returned by the raw FTS5 query — we scan into
// it then convert to entity.SearchHit.
type searchRow struct {
	Kind           string
	ID             string
	Title          string
	Snippet        string
	GroupID        string
	ConversationID *string
}

func (m *manager) SearchCards(ctx context.Context, query string, limit int) ([]*entity.SearchHit, error) {
	var rows []searchRow
	if err := m.db.FromContext(ctx).
		Raw(cardSearchSQL, query, limit).
		Scan(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrapf(err, "card search failed")
	}
	out := make([]*entity.SearchHit, 0, len(rows))
	for _, r := range rows {
		path, err := m.cardAncestorTitles(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, &entity.SearchHit{
			Kind:      r.Kind,
			ID:        r.ID,
			Title:     r.Title,
			TitlePath: path,
			Snippet:   r.Snippet,
			GroupID:   r.GroupID,
		})
	}
	return out, nil
}

func (m *manager) cardAncestorTitles(ctx context.Context, cardID string) ([]string, error) {
	var titles []string
	if err := m.db.FromContext(ctx).
		Raw(cardAncestorTitlesSQL, cardID).
		Scan(&titles).Error; err != nil {
		return nil, errors.DBFailed.Wrapf(err, "ancestor titles for card %s", cardID)
	}
	return titles, nil
}

func (m *manager) SearchMessages(ctx context.Context, query string, limit int) ([]*entity.SearchHit, error) {
	var rows []searchRow
	if err := m.db.FromContext(ctx).
		Raw(messageSearchSQL, query, limit).
		Scan(&rows).Error; err != nil {
		return nil, errors.DBFailed.Wrapf(err, "message search failed")
	}
	out := make([]*entity.SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, &entity.SearchHit{
			Kind:           r.Kind,
			ID:             r.ID,
			Title:          r.Title,
			Snippet:        r.Snippet,
			ConversationID: r.ConversationID,
		})
	}
	return out, nil
}
