//go:build sqlite_fts5

package conversation_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhanio/framingo/pkg/services/db"
	_ "github.com/xhanio/framingo/pkg/services/db/drivers/sqlite"

	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// realMigrationsDir returns the migrations source directory used by the backend.
func realMigrationsDir(t *testing.T) string {
	t.Helper()
	// pkg/services/conversation/ → repo root is three dirs up.
	return filepath.Join("..", "..", "..", "env", "default", "config", "zen-backend", "migrations")
}

func newRealDB(t *testing.T) repository.Repository {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "regression.db")
	mgr := db.New(
		db.WithType("sqlite"),
		db.WithDataSource(db.Source{DBName: tmp}),
		// version 0 → migrate to the newest file in the directory (Up()).
		// A hardcoded pin (was 13) goes stale the moment a migration is added
		// and the ORM writes a column the pinned schema lacks.
		db.WithMigration(realMigrationsDir(t), 0),
		db.WithConnection(1, 1, time.Hour, time.Hour, 30*time.Second),
	)
	if err := mgr.Init(context.Background()); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	return repository.New(mgr)
}

// TestRegression_ConversationFlow_RealSQLite walks the v0.2 conversation
// happy path on a real on-disk SQLite — the configuration that would be
// affected by the framingo v0.4.7 nested-transaction fix the v0.1 Gotcha
// card warns about.
func TestRegression_ConversationFlow_RealSQLite(t *testing.T) {
	repo := newRealDB(t)

	// Anchor entity for the cascade leg.
	groupID := ulidutil.New()
	if err := repo.CreateGroup(context.Background(), &entity.Group{
		ID: groupID, Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	svc := conversation.New(repo)
	ctx := context.Background()
	kind := "group"

	conv, err := svc.Create(ctx, "", &kind, &groupID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.AppendMessage(ctx, conv.ID, "user", "what is X?", nil); err != nil {
		t.Fatalf("AppendMessage user: %v", err)
	}
	if _, err := svc.AppendMessage(ctx, conv.ID, "assistant", "X is...", nil); err != nil {
		t.Fatalf("AppendMessage assistant: %v", err)
	}

	got, err := svc.Get(ctx, conv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "what is X?" {
		t.Fatalf("auto-title regressed: got %q", got.Title)
	}

	msgs, err := svc.GetMessages(ctx, conv.ID, 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	// Cascade leg: deleting the anchor group should drop the conversation.
	if err := svc.DeleteByAnchor(ctx, "group", groupID); err != nil {
		t.Fatalf("DeleteByAnchor: %v", err)
	}
}
