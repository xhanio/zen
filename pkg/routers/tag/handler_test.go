package tag_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	fapi "github.com/xhanio/framingo/pkg/types/api"

	"github.com/xhanio/zen/pkg/routers/tag"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	tagSvc "github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

type validatorWrap struct{ v *validator.Validate }

func (w *validatorWrap) Validate(i any) error { return w.v.Struct(i) }

func newEchoWithTagRouter(t *testing.T) (*echo.Echo, model.Tag, repository.Repository) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	svc := tagSvc.New(repo)
	r := tag.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.GET("/groups/:id/tags", api.WrapHandler(r.ListTags))
	e.PUT("/groups/:id/tags/:name", api.WrapHandler(r.RenameTag))
	e.DELETE("/groups/:id/tags/:name", api.WrapHandler(r.DeleteTag))
	return e, svc, repo
}

func seedGroup(t *testing.T, repo repository.Repository) string {
	t.Helper()
	gid := ulidutil.New()
	if err := repo.CreateGroup(context.Background(), &entity.Group{
		ID: gid, Name: "design", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return gid
}

func TestListTags_ScopedToGroup(t *testing.T) {
	e, svc, repo := newEchoWithTagRouter(t)
	gid := seedGroup(t, repo)
	if _, err := svc.EnsureByName(context.Background(), gid, "spec"); err != nil {
		t.Fatalf("seed tag: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups/"+gid+"/tags", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET got %d: %s", rec.Code, rec.Body.String())
	}
	var tags []entity.Tag
	_ = json.NewDecoder(rec.Body).Decode(&tags)
	if len(tags) != 1 || tags[0].Name != "spec" || tags[0].GroupID != gid {
		t.Fatalf("expected 1 scoped tag, got %+v", tags)
	}
}

func TestRenameTag_HTTP(t *testing.T) {
	e, svc, repo := newEchoWithTagRouter(t)
	gid := seedGroup(t, repo)
	if _, err := svc.EnsureByName(context.Background(), gid, "old"); err != nil {
		t.Fatalf("seed tag: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/groups/"+gid+"/tags/old",
		strings.NewReader(`{"new_name":"new"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Tag
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.Name != "new" {
		t.Fatalf("expected renamed 'new', got %q", got.Name)
	}
}
