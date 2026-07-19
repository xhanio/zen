package card_test

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

	"github.com/xhanio/zen/pkg/routers/card"
	cardSvc "github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

type validatorWrap struct{ v *validator.Validate }

func (w *validatorWrap) Validate(i any) error { return w.v.Struct(i) }

func newEchoWithCardRouter(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	tagSvc := tag.New(repo)
	svc := cardSvc.New(repo, tagSvc, nil)
	r := card.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/cards", api.WrapHandler(r.CreateCard))
	e.GET("/cards/:id", api.WrapHandler(r.GetCard))
	e.PUT("/cards/:id", api.WrapHandler(r.UpdateCard))
	return e, g.ID
}

func TestCreateCardWithTags_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)
	body := `{"title":"poly","group_id":"` + groupID + `","tags":["go","Rust"]}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var created entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if len(created.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", created.Tags)
	}
}

func TestCreateCardWithProvenance_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)

	// First create a parent card to satisfy the parent_card_id FK
	parentBody := `{"title":"parent","group_id":"` + groupID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(parentBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create parent: %d %s", rec.Code, rec.Body.String())
	}
	var parent entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&parent)

	// source_conversation_id validation is service-layer; without a real
	// conversation we expect 404. So this test only verifies parent_card_id
	// roundtrip; source_conversation_id is exercised by service-layer tests.
	body := `{"title":"derived","group_id":"` + groupID + `","parent_card_id":"` + parent.ID + `"}`
	req = httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var created entity.Card
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ParentCardID == nil || *created.ParentCardID != parent.ID {
		t.Fatalf("ParentCardID lost: %+v", created.ParentCardID)
	}
}

func TestUpdateCardTags_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)

	createBody := `{"title":"T","group_id":"` + groupID + `","tags":["one","two"]}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(createBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var created entity.Card
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	putBody := `{"tags":["two","three"]}`
	req = httptest.NewRequest(http.MethodPut, "/cards/"+created.ID, strings.NewReader(putBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT got %d: %s", rec.Code, rec.Body.String())
	}
	var updated entity.Card
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	got := map[string]bool{}
	for _, n := range updated.Tags {
		got[n] = true
	}
	if !got["two"] || !got["three"] || len(updated.Tags) != 2 {
		t.Fatalf("expected tags={two,three}, got %v", updated.Tags)
	}
}

func TestCreateCard_AcceptsHtmlFormat_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)
	body := `{"title":"chart","content":"<svg></svg>","format":"html","group_id":"` + groupID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.Format != "html" {
		t.Fatalf("Format = %q", got.Format)
	}
}

func TestCreateCard_DefaultsFormatToMarkdown_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)
	body := `{"title":"plain","content":"hi","group_id":"` + groupID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.Format != "markdown" {
		t.Fatalf("Format = %q", got.Format)
	}
}

func TestCreateCard_RejectsBadFormat_HTTP(t *testing.T) {
	e, groupID := newEchoWithCardRouter(t)
	body := `{"title":"x","format":"bbcode","group_id":"` + groupID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCard_LevelEntryID_HTTP(t *testing.T) {
	// Seed a group with one catalog entry and reuse the existing router setup.
	repo := repository.New(testutil.NewDB(t))
	entryID := ulidutil.New()
	g := &entity.Group{
		ID:   ulidutil.New(),
		Name: "work",
		LevelCatalog: []entity.LevelEntry{
			{ID: entryID, Weight: 0, Name: "原则"},
		},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	tagSvc := tag.New(repo)
	svc := cardSvc.New(repo, tagSvc, nil)
	r := card.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/cards", api.WrapHandler(r.CreateCard))

	body := `{"title":"x","content":"","format":"markdown","group_id":"` + g.ID + `","level_entry_id":"` + entryID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.LevelEntryID == nil || *got.LevelEntryID != entryID {
		t.Fatalf("LevelEntryID = %v want %s", got.LevelEntryID, entryID)
	}
}

func TestReorderCard_HTTP(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	parent := &entity.Card{ID: ulidutil.New(), Title: "P", GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateCard(context.Background(), parent); err != nil {
		t.Fatalf("CreateCard parent: %v", err)
	}
	pid := parent.ID
	children := make([]string, 3)
	for i := 0; i < 3; i++ {
		id := ulidutil.New()
		children[i] = id
		if err := repo.CreateCard(context.Background(), &entity.Card{
			ID: id, Title: string(rune('A' + i)),
			GroupID: g.ID, ParentCardID: &pid, Position: i,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("CreateCard child %d: %v", i, err)
		}
	}
	svc := cardSvc.New(repo, tag.New(repo), nil)
	r := card.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/cards/:id/reorder", api.WrapHandler(r.ReorderCard))

	body := `{"position":0}`
	req := httptest.NewRequest(http.MethodPost, "/cards/"+children[2]+"/reorder", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	got, _ := repo.GetCard(context.Background(), children[2])
	if got.Position != 0 {
		t.Fatalf("position after reorder = %d, want 0", got.Position)
	}
}

// newEchoWithReviewRouter mirrors newEchoWithCardRouter but also registers
// the GET /cards/:id + POST /cards/:id/review routes and returns the raw
// repo so review tests can seed cards directly.
func newEchoWithReviewRouter(t *testing.T) (*echo.Echo, repository.Repository, string) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	svc := cardSvc.New(repo, tag.New(repo), nil)
	r := card.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.GET("/cards/:id", api.WrapHandler(r.GetCard))
	e.POST("/cards/:id/review", api.WrapHandler(r.ReviewCard))
	return e, repo, g.ID
}

func TestReviewCard_HappyPath_HTTP(t *testing.T) {
	e, repo, groupID := newEchoWithReviewRouter(t)
	leaf := &entity.Card{
		ID: ulidutil.New(), Title: "leaf", Content: "body", GroupID: groupID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(context.Background(), leaf); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	body := `{"grade":"DIGESTED"}`
	req := httptest.NewRequest(http.MethodPost, "/cards/"+leaf.ID+"/review", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST review got %d: %s", rec.Code, rec.Body.String())
	}
	var got entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.ReviewGrade != "DIGESTED" {
		t.Fatalf("expected DIGESTED, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil")
	}
	if got.ReviewScore == nil || *got.ReviewScore != 50.0 {
		t.Fatalf("expected review_score 50.0, got %v", got.ReviewScore)
	}
}

func TestReviewCard_InvalidGradeReturns400(t *testing.T) {
	e, repo, groupID := newEchoWithReviewRouter(t)
	leaf := &entity.Card{
		ID: ulidutil.New(), Title: "leaf", Content: "body", GroupID: groupID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(context.Background(), leaf); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	body := `{"grade":"BOGUS"}`
	req := httptest.NewRequest(http.MethodPost, "/cards/"+leaf.ID+"/review", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetCard_TopLevelIncludesReviewScore(t *testing.T) {
	e, repo, groupID := newEchoWithReviewRouter(t)
	leaf := &entity.Card{
		ID: ulidutil.New(), Title: "leaf", Content: "body", GroupID: groupID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(context.Background(), leaf); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/cards/"+leaf.ID, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET got %d", rec.Code)
	}
	var got entity.Card
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.ReviewScore == nil {
		t.Fatalf("top-level card must include review_score")
	}
}
