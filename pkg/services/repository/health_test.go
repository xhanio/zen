package repository_test

import (
	"context"
	"testing"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
)

// probes is the health surface the supervisor calls on the repository.
// repository.Repository embeds it; asserting through this interface is what
// catches the probes being dropped from the interface.
type probes interface {
	Alive(ctx context.Context) error
	Ready(ctx context.Context) error
}

func TestProbesPassAgainstALiveDatabase(t *testing.T) {
	var p probes = repository.New(testutil.NewDB(t))
	if err := p.Alive(context.Background()); err != nil {
		t.Fatalf("Alive: %v", err)
	}
	if err := p.Ready(context.Background()); err != nil {
		t.Fatalf("Ready: %v", err)
	}
}

// The split that matters: an unreachable database must fail readiness and
// NOT liveness. Failing liveness would have the supervisor restart the
// repository, which raises no database — it would just churn.
func TestUnreachableDatabaseFailsReadinessButNotLiveness(t *testing.T) {
	db := testutil.NewDB(t)
	var p probes = repository.New(db)

	sqldb := db.DB()
	if sqldb == nil {
		t.Fatal("testutil database exposed no *sql.DB")
	}
	if err := sqldb.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	if err := p.Ready(context.Background()); err == nil {
		t.Fatal("Ready must fail once the database is unreachable")
	}
	if err := p.Alive(context.Background()); err != nil {
		t.Fatalf("Alive must survive a database outage, got: %v", err)
	}
}

// A cancelled caller must not leave a ping running; the probe inherits the
// context the supervisor's monitor hands it.
func TestReadyHonoursCallerCancellation(t *testing.T) {
	var p probes = repository.New(testutil.NewDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Ready(ctx); err == nil {
		t.Fatal("Ready must fail when its context is already cancelled")
	}
}
