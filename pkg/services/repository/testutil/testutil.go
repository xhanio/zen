// Package testutil provides an in-memory SQLite database with the M2 schema
// applied, wrapped in a minimal fake of model.Database. Used by repository
// unit tests and by any service whose tests need a real DB.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/types/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// dbCounter ensures each NewDB call gets a unique in-memory DB name so
// tests are isolated from each other while still allowing multiple
// connections within one test to see the same schema.
var dbCounter uint64

// Schema mirrors env/local/config/zen-backend/migrations/000 (core tables).
// FTS5 (migration 001) is omitted here because the in-memory SQLite used by
// repository unit tests is built without the sqlite_fts5 tag in some Go
// toolchain configurations; FTS5 behavior is exercised by integration tests
// that boot the real daemon binary (built with sqlite_fts5). Keep this const
// in sync with the production 000 migration.
const Schema = `
CREATE TABLE groups (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    rule          TEXT NOT NULL DEFAULT '',
    position      INTEGER NOT NULL DEFAULT 0,
    level_catalog TEXT NOT NULL DEFAULT '[]',
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL
);
CREATE UNIQUE INDEX groups_name_uniq ON groups(name);

CREATE TABLE tags (
    id       TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    name     TEXT NOT NULL
);
CREATE UNIQUE INDEX tags_group_name_uniq ON tags(group_id, name);
CREATE INDEX tags_group_idx ON tags(group_id);

CREATE TABLE cards (
    id                     TEXT PRIMARY KEY,
    title                  TEXT NOT NULL,
    content                TEXT NOT NULL DEFAULT '',
    summary                TEXT NOT NULL DEFAULT '',
    format                 TEXT NOT NULL DEFAULT 'markdown' CHECK (format IN ('markdown','html','text')),
    search_hint            TEXT NOT NULL DEFAULT '',
    genesis                TEXT NOT NULL DEFAULT '',
    deleted_at             DATETIME,
    group_id               TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    position               INTEGER NOT NULL DEFAULT 0,
    parent_card_id         TEXT REFERENCES cards(id) ON DELETE SET NULL,
    source_conversation_id TEXT,
    level_entry_id         TEXT,
    created_at             DATETIME NOT NULL,
    updated_at             DATETIME NOT NULL,
    review_grade           TEXT NOT NULL DEFAULT 'LGTM' CHECK (review_grade IN ('LGTM','DIGESTED','GRILLED')),
    reviewed_at            DATETIME
);
CREATE INDEX cards_group_idx ON cards(group_id);
CREATE INDEX cards_parent_card_idx ON cards(parent_card_id);
CREATE INDEX cards_source_conversation_idx ON cards(source_conversation_id);
CREATE INDEX cards_deleted_at_idx ON cards(deleted_at);
CREATE INDEX cards_level_entry_id_idx ON cards(level_entry_id);

CREATE TABLE card_tags (
    card_id TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);
CREATE INDEX card_tags_tag_idx ON card_tags(tag_id);

CREATE TABLE conversations (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    anchor_kind     TEXT,
    anchor_id       TEXT,
    created_at      DATETIME NOT NULL,
    last_message_at DATETIME NOT NULL,
    CHECK (
        (anchor_kind IS NULL AND anchor_id IS NULL)
        OR
        (anchor_kind IS NOT NULL AND anchor_id IS NOT NULL AND anchor_kind IN ('card','group'))
    )
);
CREATE INDEX conversations_anchor_idx ON conversations(anchor_kind, anchor_id);
CREATE INDEX conversations_last_message_at_idx ON conversations(last_message_at DESC);

CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    selection_text  TEXT,
    session_id      TEXT,
    session_cwd     TEXT,
    created_at      DATETIME NOT NULL,
    CHECK (role IN ('user','assistant','system')),
    CHECK (selection_text IS NULL OR role = 'user')
);
CREATE INDEX messages_conversation_idx ON messages(conversation_id, created_at);

CREATE TABLE card_references (
    id              TEXT PRIMARY KEY,
    source_card_id  TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    derived_card_id TEXT NOT NULL REFERENCES cards(id)         ON DELETE CASCADE,
    conversation_id TEXT          REFERENCES conversations(id) ON DELETE CASCADE,
    selection_text  TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    CHECK (source_card_id <> derived_card_id),
    CHECK (length(selection_text) BETWEEN 1 AND 5000)
);
CREATE INDEX card_references_source_idx       ON card_references(source_card_id);
CREATE INDEX card_references_derived_idx      ON card_references(derived_card_id);
CREATE INDEX card_references_conversation_idx ON card_references(conversation_id);
`

type fakeDB struct {
	g *gorm.DB
}

func (f *fakeDB) Name() string                       { return "test-db" }
func (f *fakeDB) Dependencies() []common.Service     { return nil }
func (f *fakeDB) Init(ctx context.Context) error     { return nil }
func (f *fakeDB) Info(w io.Writer, debug bool)       {}
func (f *fakeDB) ORM() *gorm.DB                      { return f.g }
func (f *fakeDB) DB() *sql.DB {
	d, _ := f.g.DB()
	return d
}
func (f *fakeDB) FromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return f.g.WithContext(ctx)
}

// txKey is the private context key used to propagate the tx-bound *gorm.DB
// inside Transaction's callback. Mirrors framingo's db.WrapContext pattern
// (which uses common.ContextKeyTX) at minimal scope so tests don't depend
// on framingo's private bridge.
type txKey struct{}
func (f *fakeDB) FromContextTimeout(ctx context.Context, d time.Duration) (*gorm.DB, context.CancelFunc) {
	c, cancel := context.WithTimeout(ctx, d)
	return f.g.WithContext(c), cancel
}
func (f *fakeDB) Cleanup(schema bool) error { return nil }
func (f *fakeDB) Reload() error             { return nil }
func (f *fakeDB) Transaction(ctx context.Context, fn func(context.Context) error, opts ...*sql.TxOptions) error {
	// If ctx already carries an active tx, start a nested transaction on
	// it (gorm uses savepoints for nesting). Otherwise start from root.
	base := f.g
	if existing, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		base = existing
	}
	return base.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// NewDB opens an in-memory SQLite with a per-call unique name, applies the
// schema, and returns it as a model.Database suitable for passing to
// repository.New(...).
//
// The DSN uses `cache=shared` plus a unique per-call name so all connections
// the gorm pool checks out (including transaction sessions) see the same
// in-memory database, while tests stay isolated from each other.
func NewDB(t *testing.T) model.Database {
	t.Helper()
	name := fmt.Sprintf("zen_test_%d", atomic.AddUint64(&dbCounter, 1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", name)
	g, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := g.Exec(Schema).Error; err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	// Hold one connection open for the duration of the test. SQLite's
	// shared-cache in-memory DB is freed when the last connection closes;
	// without this anchor, transactions that briefly drop to zero connections
	// would lose the schema. Cleanup releases it.
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
