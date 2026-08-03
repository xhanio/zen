package conversation

import (
	"time"

	fmodel "github.com/xhanio/framingo/pkg/types/model"
	"github.com/xhanio/framingo/pkg/utils/log"

	"github.com/xhanio/zen/pkg/types/model"
)

// Test-only surface. This file is part of package conversation but compiles
// only under `go test`, so the external conversation_test package can reach
// the concrete router while nothing here ships in a production binary.

// RouterForTest exposes the unexported router to the external test package.
type RouterForTest = router

// Every constructor here supplies a real logger. StreamWS logs on the
// registration-failure path and Handlers() logs its handler count, and a nil
// logger would panic in either — a test-only crash in code the tests exist to
// exercise.
func NewForTest(svc model.Conversation) *RouterForTest {
	return newRouter(svc, nil, nil, nil, log.Default)
}

func NewForTestWithBus(svc model.Conversation, bus fmodel.MessageBus) *RouterForTest {
	return newRouter(svc, bus, nil, nil, log.Default)
}

func NewForTestWithPresence(svc model.Conversation, bus fmodel.MessageBus, pres model.Presence) *RouterForTest {
	return newRouter(svc, bus, pres, nil, log.Default)
}

func NewForTestWithDelivery(svc model.Conversation, bus fmodel.MessageBus, pres model.Presence, del model.Delivery) *RouterForTest {
	return newRouter(svc, bus, pres, del, log.Default)
}

// SetRegistrationTimeout shrinks the silence budget for tests that assert on an
// unregistered socket. Set it before the server starts; it is read by the
// handler goroutine.
func (r *RouterForTest) SetRegistrationTimeout(d time.Duration) { r.regTimeout = d }

// SetLogger swaps the logger so a test can read what the handler actually
// emitted. Set it before the server starts.
func (r *RouterForTest) SetLogger(l log.Logger) { r.log = l }
