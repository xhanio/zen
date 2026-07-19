package group_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	fapi "github.com/xhanio/framingo/pkg/types/api"

	"github.com/xhanio/zen/pkg/routers/group"
	groupSvc "github.com/xhanio/zen/pkg/services/group"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type validatorWrap struct{ v *validator.Validate }

func (w *validatorWrap) Validate(i any) error { return w.v.Struct(i) }

func newEchoWithRouter(t *testing.T) *echo.Echo {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	svc := groupSvc.New(repo, nil)
	r := group.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	// Wire framingo's xhanio/errors → HTTP status mapping so handler unit
	// tests see the same response codes the real server returns.
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/groups", api.WrapHandler(r.CreateGroup))
	e.GET("/groups", api.WrapHandler(r.ListGroups))
	e.GET("/groups/:id", api.WrapHandler(r.GetGroup))
	return e
}

func TestCreateGroup_WithRule(t *testing.T) {
	e := newEchoWithRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/groups",
		strings.NewReader(`{"name":"design","rule":"Chinese + HTML."}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var created entity.Group
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Rule != "Chinese + HTML." {
		t.Fatalf("got rule %q", created.Rule)
	}
}

func TestCreateAndListGroups_HTTP(t *testing.T) {
	e := newEchoWithRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/groups",
		strings.NewReader(`{"name":"work"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var created entity.Group
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" || created.Name != "work" {
		t.Fatalf("bad created body: %+v", created)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/groups", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET got %d: %s", rec2.Code, rec2.Body.String())
	}
	var list []entity.Group
	if err := json.NewDecoder(rec2.Body).Decode(&list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("bad list response: %+v", list)
	}
}

func TestCreateGroup_EmptyName_Returns400(t *testing.T) {
	e := newEchoWithRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/groups",
		strings.NewReader(`{"name":""}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateGroup_ReplaceCatalog_HTTP(t *testing.T) {
	e := newEchoWithRouter(t)
	repo := repository.New(testutil.NewDB(t))
	svc := groupSvc.New(repo, nil)
	r := group.NewForTest(svc)
	e2 := echo.New()
	e2.Validator = &validatorWrap{v: validator.New()}
	e2.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e2.POST("/groups", api.WrapHandler(r.CreateGroup))
	e2.PUT("/groups/:id", api.WrapHandler(r.UpdateGroup))
	_ = e // keep helper used for shared setup

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"design"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e2.ServeHTTP(rec, req)
	var g entity.Group
	_ = json.NewDecoder(rec.Body).Decode(&g)

	body := `{"level_catalog":[{"number":0,"name":"原则"},{"number":1,"name":"决策"}]}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/groups/"+g.ID, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e2.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Group
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if len(got.LevelCatalog) != 2 || got.LevelCatalog[0].Name != "原则" {
		t.Fatalf("catalog: %+v", got.LevelCatalog)
	}
}
