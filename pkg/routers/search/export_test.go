package search

import (
	"github.com/xhanio/zen/pkg/types/model"
)

// Test-only surface. This file is part of package search but compiles only
// under `go test`, so the external search_test package can reach the concrete
// router while nothing here ships in a production binary.

// RouterForTest exposes the unexported router to the external test package.
type RouterForTest = router

// NewForTest constructs a router for unit tests without requiring a logger.
// The handlers under test never log; Handlers() would, so tests calling it
// must go through New instead.
func NewForTest(svc model.Search) *RouterForTest {
	return newRouter(svc, nil)
}
