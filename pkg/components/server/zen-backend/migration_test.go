//go:build sqlite_fts5

package zenbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// bootOnce starts a daemon against an existing config and returns a stop func.
// Two calls against the same config (and therefore the same DB file) simulate a
// restart of a deployed backend.
func bootOnce(t *testing.T, cfgPath string, port int) func() {
	t.Helper()
	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.Init(ctx); err != nil {
		cancel()
		t.Fatalf("Init: %v", err)
	}
	go func() {
		if err := srv.Start(ctx); err != nil {
			t.Logf("Start returned: %v", err)
		}
	}()
	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/readyz", 10*time.Second)
	return func() {
		_ = srv.Stop(true)
		cancel()
		// Give the listener a moment to release the port before the next boot.
		time.Sleep(200 * time.Millisecond)
	}
}

// A restart must not touch data. The schema version a binary drives the DB to
// is whatever its own migrations directory contains, and golang-migrate walks
// DOWN as readily as up: any fixed target below the current version silently
// runs the down-migrations. 017_add_review_grade.down.sql drops the column, so
// a downgrade-then-upgrade cycle resets every grade to the 'LGTM' default —
// invisibly, because LGTM is also the never-graded state.
func TestDaemonRestart_PreservesReviewGrades(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-restart.db")
	cfgPath := writeTestConfig(t, dbPath, port)
	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)

	stop := bootOnce(t, cfgPath, port)

	// A group, a card, and a grade that is not the default.
	gResp, err := http.Post(base+"/groups", "application/json",
		strings.NewReader(`{"name":"g"}`))
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	var g struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&g)
	gResp.Body.Close()

	cResp, err := http.Post(base+"/cards", "application/json",
		strings.NewReader(fmt.Sprintf(`{"title":"t","content":"c","group_id":%q}`, g.ID)))
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	var card struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(cResp.Body).Decode(&card)
	cResp.Body.Close()
	if card.ID == "" {
		t.Fatal("no card id")
	}

	rResp, err := http.Post(base+"/cards/"+card.ID+"/review", "application/json",
		strings.NewReader(`{"grade":"GRILLED"}`))
	if err != nil {
		t.Fatalf("review card: %v", err)
	}
	if rResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(rResp.Body)
		rResp.Body.Close()
		t.Fatalf("review status %d: %s", rResp.StatusCode, b)
	}
	rResp.Body.Close()

	stop()

	// ---- restart on the same database ----
	stop2 := bootOnce(t, cfgPath, port)
	t.Cleanup(stop2)

	getResp, err := http.Get(base + "/cards/" + card.ID)
	if err != nil {
		t.Fatalf("get card after restart: %v", err)
	}
	defer getResp.Body.Close()
	var after struct {
		ReviewGrade string  `json:"review_grade"`
		ReviewedAt  *string `json:"reviewed_at"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&after); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if after.ReviewGrade != "GRILLED" {
		t.Fatalf("restart wiped the grade: review_grade = %q, want GRILLED", after.ReviewGrade)
	}
	if after.ReviewedAt == nil {
		t.Fatal("restart wiped reviewed_at")
	}
}
