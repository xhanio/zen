package search_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	fapi "github.com/xhanio/framingo/pkg/types/api"

	"github.com/xhanio/zen/pkg/routers/search"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	searchSvc "github.com/xhanio/zen/pkg/services/search"
	"github.com/xhanio/zen/pkg/types/api"
)

type validatorWrap struct{ v *validator.Validate }

func (w *validatorWrap) Validate(i any) error { return w.v.Struct(i) }

func newEchoWithSearchRouter(t *testing.T) *echo.Echo {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	svc := searchSvc.New(repo)
	r := search.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.GET("/search", api.WrapHandler(r.Search))
	return e
}

func TestSearch_MissingQuery_Returns400(t *testing.T) {
	e := newEchoWithSearchRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing q, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearch_BadScope_Returns400(t *testing.T) {
	e := newEchoWithSearchRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/search?q=foo&scope=bogus", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
