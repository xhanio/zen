// Package api defines this project's request-context wrapper around
// echo.Context and the glue that lets routers declare handlers as
// `func(c Context) error` instead of `func(c echo.Context) error`.
//
// Why this layer exists:
//   - One context value satisfies both echo.Context (request/response,
//     binding) and context.Context (deadline/cancellation propagation
//     into services), so handlers can pass `c` straight through.
//   - Project-wide accessors (credential, session, trace-id) and
//     custom binders live here, so adding one doesn't ripple through
//     every handler signature.
//
// The framework's API server only accepts echo.HandlerFunc /
// func(echo.Context, *websocket.Conn) error. WrapHandler /
// WrapWebSocket / DiscoverHandlers bridge our richer signatures back
// to those at router registration time.
package api

import (
	"context"
	"reflect"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/xhanio/framingo/pkg/types/api"
)

// Context is the per-request value passed to router handlers.
// It embeds echo.Context (so handlers keep all of Echo's API) and
// context.Context (so handlers can hand `c` to any context-aware
// service call without unwrapping to c.Request().Context() first).
type Context interface {
	// echo wrap: echo.Context for request/response handling,
	// context.Context for cancellation/deadline propagation, plus
	// thin helpers over Echo's parameter binders.
	echo.Context
	context.Context
	BindQuery() *echo.ValueBinder
	BindPath() *echo.ValueBinder
	BindForm() *echo.ValueBinder
	BindAny(i any) error

	// helpers: typed accessors for values stashed into the echo
	// context by middleware (auth, tracing). They return (_, false)
	// when the value is missing or the wrong type, so handlers don't
	// have to repeat the assertion dance.
	TraceID() (string, bool)
}

// ctx is the default Context implementation: an echo.Context with
// context.Context semantics layered on by deferring to the underlying
// *http.Request's context for Deadline/Done/Err, and resolving Value
// lookups against echo's request-scoped Get store before that context.
type ctx struct {
	echo.Context
}

func (c *ctx) Deadline() (time.Time, bool) {
	return c.Request().Context().Deadline()
}

func (c *ctx) Done() <-chan struct{} {
	return c.Request().Context().Done()
}

func (c *ctx) Err() error {
	return c.Request().Context().Err()
}

// Value resolves echo's request-scoped store first, then falls through to the
// request's own context. Both halves are needed. Framingo's middlewares stash
// their values (_trace, _credential, ...) in echo's store under string keys, so
// the store must win for those. Everything else — a stdlib http middleware
// wrapping the handler, an otel span, anything keyed by an unexported struct
// type as Go idiom prescribes — lives on the request context, and returning nil
// for it would make those values vanish with no error to explain it.
func (c *ctx) Value(key any) any {
	if k, ok := key.(string); ok {
		if v := c.Get(k); v != nil {
			return v
		}
	}
	return c.Request().Context().Value(key)
}

func (c *ctx) TraceID() (string, bool) {
	traceID := c.Get(api.ContextKeyTrace)
	if traceID == nil {
		return "", false
	}
	tid, ok := traceID.(string)
	return tid, ok
}

func (c *ctx) BindQuery() *echo.ValueBinder {
	return echo.QueryParamsBinder(c.Context)
}

func (c *ctx) BindPath() *echo.ValueBinder {
	return echo.PathParamsBinder(c.Context)
}

func (c *ctx) BindForm() *echo.ValueBinder {
	return echo.FormFieldBinder(c.Context)
}

func (c *ctx) BindAny(i any) error {
	return c.Context.Bind(i)
}

// WrapContext adapts an echo.Context into a Context. Router code gets this for
// free via WrapHandler/WrapWebSocket; call it directly only when wiring a
// handler onto a bare echo instance, as the tests do.
//
// The result must not outlive the handler that received it. Echo pools its
// contexts and resets them once the handler returns, and Deadline/Done/Err/Value
// here read through to c on every call — so a Context held by a goroutine that
// outlives the request ends up reading whichever request recycled that slot.
// (`c.Request().Context()`, which this replaced, stayed valid because it was a
// value captured once rather than a live read.) To carry cancellation into a
// goroutine, derive before returning — `ctx, cancel := context.WithCancel(c)` —
// which reads the parent once at construction; that is what the WebSocket
// handlers do. To outlive the request deliberately, detach with
// context.WithoutCancel(c) rather than passing c on.
func WrapContext(c echo.Context) Context {
	return &ctx{c}
}

// HandlerFunc / WebSocketHandlerFunc are the recommended handler signatures
// for routers in this project. WrapHandler / WrapWebSocket adapt them
// into the echo signatures the framework's server registers.

type WebSocketHandlerFunc func(Context, *websocket.Conn) error

func WrapWebSocket(wf WebSocketHandlerFunc) func(echo.Context, *websocket.Conn) error {
	return func(c echo.Context, conn *websocket.Conn) error {
		return wf(WrapContext(c), conn)
	}
}

type HandlerFunc func(Context) error

func WrapHandler(hf HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return hf(WrapContext(c))
	}
}

// DiscoverHandlers reflects over r's methods and returns those matching a
// known handler signature, keyed by method name. It lets a router expose
//
//	func (r *router) Handlers() map[string]any { return api.DiscoverHandlers(r) }
//
// instead of hand-listing every method, and transparently wraps
// `func(Context) error` / `func(Context, *websocket.Conn) error` methods
// into the echo-compatible signatures the framework expects. Methods named
// "Handlers" are skipped so the router's own Handlers() doesn't recurse.
func DiscoverHandlers(r any) map[string]any {
	handlers := make(map[string]any)
	rv := reflect.ValueOf(r)
	rt := reflect.TypeOf(r)
	for i := 0; i < rt.NumMethod(); i++ {
		method := rt.Method(i)
		if method.Name == "Handlers" {
			continue
		}
		switch fn := rv.Method(i).Interface().(type) {
		case func(echo.Context) error:
			handlers[method.Name] = echo.HandlerFunc(fn)
		case func(Context) error:
			handlers[method.Name] = WrapHandler(fn)
		case func(echo.Context, *websocket.Conn) error:
			handlers[method.Name] = fn
		case func(Context, *websocket.Conn) error:
			handlers[method.Name] = WrapWebSocket(fn)
		}
	}
	return handlers
}
