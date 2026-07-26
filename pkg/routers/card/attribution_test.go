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

// Exercises the real router over a real service, so the assertion covers the
// whole path: header → handler → service → stored snapshot.
func newAttributionCtx(t *testing.T) (*echo.Echo, repository.Repository, string) {
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
	e.POST("/cards", api.WrapHandler(r.CreateCard))
	e.PUT("/cards/:id", api.WrapHandler(r.UpdateCard))
	return e, repo, g.ID
}

func createCardHTTP(t *testing.T, e *echo.Echo, groupID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/cards",
		strings.NewReader(`{"title":"c","content":"before","group_id":"`+groupID+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create card: status %d body %s", rec.Code, rec.Body.String())
	}
	var out entity.Card
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode card: %v", err)
	}
	return out.ID
}

func latestSnapshot(t *testing.T, repo repository.Repository, cardID string) *entity.CardSnapshot {
	t.Helper()
	got, err := repo.ListCardSnapshots(context.Background(),
		api.ListSnapshotsRequest{CardID: &cardID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("no snapshots for card %s", cardID)
	}
	return got[0] // newest-first
}

func TestUpdateCard_AgentHeaderAndConversationReachTheSnapshot(t *testing.T) {
	e, repo, groupID := newAttributionCtx(t)
	cardID := createCardHTTP(t, e, groupID)
	conv := ulidutil.New()

	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID,
		strings.NewReader(`{"content":"after","conversation_id":"`+conv+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(api.ActorHeader, "agent")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status %d body %s", rec.Code, rec.Body.String())
	}

	snap := latestSnapshot(t, repo, cardID)
	if snap.Actor != "agent" {
		t.Fatalf("actor = %q, want agent", snap.Actor)
	}
	if snap.ConversationID == nil || *snap.ConversationID != conv {
		t.Fatalf("conversation = %v, want %q", snap.ConversationID, conv)
	}
}

func TestUpdateCard_NoHeaderIsAUserEdit(t *testing.T) {
	e, repo, groupID := newAttributionCtx(t)
	cardID := createCardHTTP(t, e, groupID)

	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID,
		strings.NewReader(`{"content":"after"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status %d body %s", rec.Code, rec.Body.String())
	}

	snap := latestSnapshot(t, repo, cardID)
	if snap.Actor != "user" {
		t.Fatalf("actor = %q, want user", snap.Actor)
	}
	if snap.ConversationID != nil {
		t.Fatalf("conversation should be nil for a hand edit, got %q", *snap.ConversationID)
	}
}

// An arbitrary actor value must not be trusted through: anything that is not
// exactly "agent" is a user edit.
func TestUpdateCard_UnknownActorFallsBackToUser(t *testing.T) {
	e, repo, groupID := newAttributionCtx(t)
	cardID := createCardHTTP(t, e, groupID)

	req := httptest.NewRequest(http.MethodPut, "/cards/"+cardID,
		strings.NewReader(`{"content":"after"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(api.ActorHeader, "root")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status %d body %s", rec.Code, rec.Body.String())
	}

	if got := latestSnapshot(t, repo, cardID).Actor; got != "user" {
		t.Fatalf("actor = %q, want user", got)
	}
}
