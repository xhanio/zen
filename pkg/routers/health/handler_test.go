package health

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	"github.com/xhanio/zen/pkg/types/api"
)

// minimalDB lets us reuse the framingo db.Manager interface where only DB() is exercised.
type fakeDB struct {
	sqldb *sql.DB
}

func (f *fakeDB) DB() *sql.DB { return f.sqldb }

func TestHealthZ_AlwaysOK(t *testing.T) {
	r := &router{db: nil, log: nil} // HealthZ ignores db; nil is fine for liveness probe.

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := r.HealthZ(api.WrapContext(c)); err != nil {
		t.Fatalf("HealthZ returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestReadyZ_OKWhenDBPings(t *testing.T) {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	defer sqldb.Close()

	r := &router{db: &fakeDB{sqldb: sqldb}, log: nil}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := r.ReadyZ(api.WrapContext(c)); err != nil {
		t.Fatalf("ReadyZ returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ready"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestReadyZ_ErrWhenDBClosed(t *testing.T) {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	sqldb.Close() // force a failing Ping.

	r := &router{db: &fakeDB{sqldb: sqldb}, log: nil}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := r.ReadyZ(api.WrapContext(c)); err == nil {
		t.Fatalf("expected ReadyZ to return error, got nil")
	}
}
