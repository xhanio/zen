package repository

import (
	"context"
	"time"

	"github.com/xhanio/errors"
)

// RunV12Backfill applies the v0.12 auto-upgrade rule to every existing card.
// For each card, count role="user" messages across its card-anchored
// conversations; raise review_grade to DIGESTED (>=1) or GRILLED (>=3);
// set reviewed_at to the most recent qualifying message's created_at.
//
// Idempotent: cards already at or above the computed floor are left alone.
// Safe to run on every startup — cheap enough for typical Zen sizes.
func (m *manager) RunV12Backfill(ctx context.Context) error {
	// SQLite returns MAX(created_at) as a string; scan into a string field
	// and parse it. Postgres would tolerate time.Time here but sqlite3's
	// driver.Value stays as text.
	type stat struct {
		CardID    string `gorm:"column:card_id"`
		UserCount int    `gorm:"column:user_count"`
		LastAt    string `gorm:"column:last_at"`
	}
	var stats []stat
	err := m.db.FromContext(ctx).Raw(`
        SELECT c.anchor_id AS card_id,
               COUNT(msg.id) AS user_count,
               MAX(msg.created_at) AS last_at
        FROM conversations c
        JOIN messages msg ON msg.conversation_id = c.id
        WHERE c.anchor_kind = 'card' AND msg.role = 'user'
        GROUP BY c.anchor_id
    `).Scan(&stats).Error
	if err != nil {
		return errors.DBFailed.Wrap(err)
	}
	for _, s := range stats {
		var floor string
		switch {
		case s.UserCount >= 3:
			floor = "GRILLED"
		case s.UserCount >= 1:
			floor = "DIGESTED"
		default:
			continue
		}
		card, err := m.GetCard(ctx, s.CardID)
		if err != nil {
			continue // card missing/trashed — skip silently
		}
		if card.DeletedAt != nil {
			continue
		}
		if gradeRankV12(card.ReviewGrade) >= gradeRankV12(floor) {
			continue // already at or above
		}
		card.ReviewGrade = floor
		if t, ok := parseSQLiteDatetime(s.LastAt); ok {
			card.ReviewedAt = &t
		}
		if err := m.UpdateCard(ctx, card); err != nil {
			return errors.Wrap(err)
		}
	}
	return nil
}

// parseSQLiteDatetime accepts the common SQLite datetime string encodings and
// returns a time.Time. sqlite3 drivers commonly emit "2006-01-02 15:04:05" or
// RFC 3339. Returns (_, false) if the string is empty or unparseable.
func parseSQLiteDatetime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// gradeRankV12 mirrors conversation.gradeRank; kept package-local to avoid a
// cross-service dependency.
func gradeRankV12(g string) int {
	switch g {
	case "LGTM":
		return 0
	case "DIGESTED":
		return 1
	case "GRILLED":
		return 2
	default:
		return -1
	}
}
