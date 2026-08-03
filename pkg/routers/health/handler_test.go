package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/types/entity"

	"github.com/xhanio/zen/pkg/types/api"
)

// fakeSupervisor stands in for supervisor.Manager, of which this router uses
// only the two verdicts and the stats behind the readiness body.
type fakeSupervisor struct {
	alive error
	ready error
	stats []*entity.SupervisorStats
}

func (f *fakeSupervisor) Alive(context.Context) error { return f.alive }
func (f *fakeSupervisor) Ready(context.Context) error { return f.ready }
func (f *fakeSupervisor) Stats() ([]*entity.SupervisorStats, error) {
	return f.stats, nil
}

func serve(t *testing.T, h func(api.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	if err := h(api.WrapContext(c)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	return rec
}

func TestHealthzOKWhileSupervisorIsAlive(t *testing.T) {
	r := newRouter(&fakeSupervisor{}, nil)
	rec := serve(t, r.Healthz)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

// A dependency outage must NOT fail liveness — restarting the pod raises no
// database. Liveness goes red only once the supervisor's recovery is spent.
func TestHealthzStaysOKWhenOnlyReadinessIsFailing(t *testing.T) {
	r := newRouter(&fakeSupervisor{ready: errors.Newf("database ping failed")}, nil)
	if rec := serve(t, r.Healthz); rec.Code != http.StatusOK {
		t.Fatalf("readiness failure must not fail liveness; got %d", rec.Code)
	}
}

func TestHealthz503WhenRecoveryIsSpent(t *testing.T) {
	r := newRouter(&fakeSupervisor{alive: errors.Newf("restarts exhausted")}, nil)
	rec := serve(t, r.Healthz)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "restarts exhausted") {
		t.Fatalf("body should carry the reason: %s", rec.Body.String())
	}
}

func TestReadyzOKWhenEveryServiceIsReady(t *testing.T) {
	r := newRouter(&fakeSupervisor{}, nil)
	rec := serve(t, r.Readyz)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ready":true`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestReadyz503ItemizesTheOffenders(t *testing.T) {
	r := newRouter(&fakeSupervisor{
		ready: errors.Newf("repository not ready"),
		stats: []*entity.SupervisorStats{
			{Name: "zed", Ready: false, ReadinessErr: errors.Newf("database ping failed")},
			{Name: "alpha", Ready: true},
			{Name: "beta", Ready: false},
		},
	}, nil)
	rec := serve(t, r.Readyz)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "alpha") {
		t.Fatalf("ready services must not be listed: %s", body)
	}
	// Sorted by name, so beta precedes zed regardless of stats order.
	if strings.Index(body, "beta") > strings.Index(body, "zed") {
		t.Fatalf("offenders should be sorted by name: %s", body)
	}
	if !strings.Contains(body, "database ping failed") {
		t.Fatalf("the specific readiness error should survive: %s", body)
	}
	if !strings.Contains(body, `"error":"not ready"`) {
		t.Fatalf("a bare not-ready service needs a fallback detail: %s", body)
	}
}
