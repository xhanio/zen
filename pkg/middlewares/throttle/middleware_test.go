package throttle

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/xhanio/framingo/pkg/types/api"
)

// attach builds one enabled attachment and wraps a handler that always
// succeeds, so a non-nil error out of the chain means the request was rate
// limited.
func attach(t *testing.T, config []byte) echo.HandlerFunc {
	t.Helper()
	fn, err := New().Func(true, config)
	if err != nil {
		t.Fatalf("Func: %v", err)
	}
	if fn == nil {
		t.Fatal("Func declined the attachment")
	}
	return fn(func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
}

// call issues one request through h as if the Info middleware had already run,
// which is what puts the resolved client IP on the context.
func call(h echo.HandlerFunc, ip string) error {
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	c.Set(api.ContextKeyRequestInfo, &api.RequestInfo{IP: ip})
	return h(c)
}

func TestNameIsThrottle(t *testing.T) {
	// The name is what router.yaml attaches by and what the server looks the
	// config block up under, so a package rename would silently detach it.
	if name := New().Name(); name != "throttle" {
		t.Fatalf("Name() = %q, want \"throttle\"", name)
	}
}

func TestNoLimitPassesEverything(t *testing.T) {
	// A router may attach throttle unconditionally; with no limit configured
	// anywhere the attachment has to be transparent rather than refuse.
	for _, config := range [][]byte{nil, []byte("rps: 0\nburst_size: 200\n"), []byte("rps: 100.0\nburst_size: 0\n")} {
		h := attach(t, config)
		for i := 0; i < 50; i++ {
			if err := call(h, "10.0.0.1"); err != nil {
				t.Fatalf("config %q limited request %d: %v", config, i, err)
			}
		}
	}
}

func TestInvalidConfigFails(t *testing.T) {
	// A bad limit must fail startup rather than leave the route unthrottled.
	if _, err := New().Func(true, []byte("rps: not-a-number\n")); err == nil {
		t.Fatal("Func accepted a non-numeric rps")
	}
}

// A route that switches the middleware off (`- throttle: false`) must get no
// function at all, not a limiter that happens to allow everything — and the
// switch has to win even when a limit is inherited from the group or server.
func TestDisabledDeclinesTheAttachment(t *testing.T) {
	for _, config := range [][]byte{nil, []byte("rps: 100.0\nburst_size: 200\n")} {
		fn, err := New().Func(false, config)
		if err != nil {
			t.Fatalf("Func(false, %q): %v", config, err)
		}
		if fn != nil {
			t.Fatalf("Func(false, %q) attached anyway", config)
		}
	}
}

func TestMissingRequestInfoIsAnError(t *testing.T) {
	// The middleware belongs downstream of Info. Attached where Info has not
	// run there is no IP to key on, and passing the request through would be
	// an unthrottled route that looks configured.
	h := attach(t, []byte("rps: 1.0\nburst_size: 1\n"))
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	if err := h(c); err == nil {
		t.Fatal("request with no RequestInfo was served")
	}
}

func TestBurstIsAllowedThenLimited(t *testing.T) {
	h := attach(t, []byte("rps: 100.0\nburst_size: 200\n"))
	var allowed, denied int
	for i := 0; i < 250; i++ {
		if err := call(h, "10.0.0.1"); err != nil {
			denied++
		} else {
			allowed++
		}
	}
	// The bucket starts full at burst_size and refills at rps; a tight loop
	// finishes far inside a second, so the refill contributes nothing.
	if allowed < 200 {
		t.Fatalf("allowed %d of a 200 burst", allowed)
	}
	if denied == 0 {
		t.Fatal("250 requests against a burst of 200 were never limited")
	}
}

func TestBucketsAreSeparatePerIP(t *testing.T) {
	h := attach(t, []byte("rps: 1.0\nburst_size: 1\n"))
	if err := call(h, "10.0.0.1"); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if err := call(h, "10.0.0.1"); err == nil {
		t.Fatal("second request from the same client was not limited")
	}
	if err := call(h, "10.0.0.2"); err != nil {
		t.Fatalf("another client shares a bucket: %v", err)
	}
}

func TestAttachmentsDoNotShareLimiters(t *testing.T) {
	// Func is called once per attachment point, and each route's table lives
	// in its own closure - one exhausted route must not limit another.
	first := attach(t, []byte("rps: 1.0\nburst_size: 1\n"))
	second := attach(t, []byte("rps: 1.0\nburst_size: 1\n"))
	if err := call(first, "10.0.0.1"); err != nil {
		t.Fatalf("first attachment: %v", err)
	}
	if err := call(first, "10.0.0.1"); err == nil {
		t.Fatal("first attachment did not limit its second request")
	}
	if err := call(second, "10.0.0.1"); err != nil {
		t.Fatalf("second attachment shares the first's limiter: %v", err)
	}
}
