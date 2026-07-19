package repo

import "context"

type CardTag interface {
	AttachTag(ctx context.Context, cardID, tagID string) error
	DetachTag(ctx context.Context, cardID, tagID string) error
	ListTagsForCard(ctx context.Context, cardID string) ([]string, error)
	ListCardsForTag(ctx context.Context, tagName string) ([]string, error)
}
