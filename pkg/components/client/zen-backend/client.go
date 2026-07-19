package zenbackend

import (
	"context"
	"encoding/json"

	"github.com/labstack/echo/v4"
	"github.com/xhanio/errors"
	framingoClient "github.com/xhanio/framingo/pkg/services/api/client"
	"github.com/xhanio/framingo/pkg/types/api"
	"github.com/xhanio/framingo/pkg/types/common"
)

type client struct {
	framingo framingoClient.Client
	// fopts is the staged set of framingo options collected from zen Option
	// callbacks before framingoClient.New is called. Cleared after New
	// returns.
	fopts []framingoClient.Option
	// token is captured from WithAuthToken and applied via SetHeaders after
	// framingo's Init succeeds.
	token string
}

// New returns a Client pointed at baseURL. baseURL must include the API
// prefix (e.g. "http://127.0.0.1:8080/api/v1").
func New(baseURL string, opts ...Option) Client {
	c := &client{}
	for _, opt := range opts {
		opt(c)
	}
	c.framingo = framingoClient.New(baseURL, c.fopts...)
	if err := c.framingo.Init(context.Background()); err != nil {
		// Init only fails when no endpoint is supplied; we always supply one.
		panic(errors.Wrap(err))
	}
	if c.token != "" {
		c.framingo.SetHeaders(common.NewPair("Authorization", "Bearer "+c.token))
	}
	c.fopts = nil
	return c
}

func (c *client) Group() GroupClient               { return &groupClient{c: c} }
func (c *client) Tag() TagClient                   { return &tagClient{c: c} }
func (c *client) Card() CardClient                 { return &cardClient{c: c} }
func (c *client) Search() SearchClient             { return &searchClient{c: c} }
func (c *client) Conversation() ConversationClient { return &conversationClient{c: c} }
func (c *client) Reference() ReferenceClient       { return &referenceClient{c: c} }

// doJSON sends a request through the framingo HTTP client. body may be nil
// for methods without a request body; out may be nil for endpoints that
// return no content. Backend errors are decoded into *api.ErrorBody and
// wrapped with the xhanio/errors category named by ErrorBody.Kind, so
// callers can use errors.Is(err, errors.NotFound) etc.
func (c *client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	req := &framingoClient.Request{
		Method: method,
		Path:   path,
	}
	if body != nil {
		req.ContentType = echo.MIMEApplicationJSON
		req.Body = body
	}
	resp, err := c.framingo.Send(ctx, req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return categorize(err)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return errors.Internal.Wrapf(err, "decode response body")
		}
	}
	return nil
}

// doNoContent is doJSON for endpoints that return 204 No Content on success.
func (c *client) doNoContent(ctx context.Context, method, path string, body any) error {
	return c.doJSON(ctx, method, path, body, nil)
}

// categorize lifts a framingo HTTP error into an xhanio/errors-categorized
// error so callers can write errors.Is(err, errors.NotFound). The category
// comes from the wire (api.ErrorBody.Kind), looked up via LookupCategory.
// Unknown kinds pass the *api.ErrorBody through unchanged; non-ErrorBody
// errors (transport failures, etc.) are wrapped without a category.
func categorize(err error) error {
	if eb, ok := err.(*api.ErrorBody); ok {
		if cat := errors.LookupCategory(eb.Kind); cat != nil {
			return cat.Wrap(eb)
		}
		return eb
	}
	return errors.Wrap(err)
}
