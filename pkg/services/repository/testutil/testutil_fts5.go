//go:build sqlite_fts5

package testutil

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/xhanio/framingo/pkg/types/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SchemaFTS5 mirrors env/default/config/zen-backend/migrations/010 — the
// cards_fts virtual table + sync triggers that skip soft-deleted rows.
// Applied after the core Schema in NewDBWithFTS5.
// The second FTS column is named search_hint (not content) on purpose: with an
// external-content table, snippet()/highlight() reconstruct text from the cards
// column of the same name, so the column FTS reads back must be the one the
// triggers index — search_hint (the stripped text) — or snippets leak raw HTML.
const SchemaFTS5 = `
CREATE VIRTUAL TABLE cards_fts USING fts5(
    title,
    search_hint,
    content='cards',
    content_rowid='rowid',
    tokenize='unicode61'
);
CREATE TRIGGER cards_ai AFTER INSERT ON cards
WHEN new.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(rowid, title, search_hint)
    VALUES (new.rowid, new.title, new.search_hint);
END;
CREATE TRIGGER cards_ad AFTER DELETE ON cards
WHEN old.deleted_at IS NULL BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, search_hint)
    VALUES('delete', old.rowid, old.title, old.search_hint);
END;
CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, search_hint)
    SELECT 'delete', old.rowid, old.title, old.search_hint
    WHERE old.deleted_at IS NULL;
    INSERT INTO cards_fts(rowid, title, search_hint)
    SELECT new.rowid, new.title, new.search_hint
    WHERE new.deleted_at IS NULL;
END;
`

// NewDBWithFTS5 opens an in-memory SQLite with both the core schema and the
// FTS5 virtual table + triggers applied. Returns model.Database; tests that
// only need core tables should use NewDB(t) instead.
//
// REQUIRES the test binary to be built with `-tags sqlite_fts5`.
func NewDBWithFTS5(t *testing.T) model.Database {
	t.Helper()
	name := fmt.Sprintf("zen_fts5_test_%d", atomic.AddUint64(&dbCounter, 1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", name)
	g, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := g.Exec(Schema).Error; err != nil {
		t.Fatalf("apply core schema: %v", err)
	}
	if err := g.Exec(SchemaFTS5).Error; err != nil {
		t.Fatalf("apply FTS5 schema: %v", err)
	}
	sqlDB, err := g.DB()
	if err != nil {
		t.Fatalf("acquire sql.DB: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("anchor connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &fakeDB{g: g}
}
