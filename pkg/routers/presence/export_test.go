package presence

import (
	"github.com/xhanio/framingo/pkg/utils/log"

	"github.com/xhanio/zen/pkg/types/model"
)

// Test-only surface. This file is part of package presence but compiles only
// under `go test`, so the external presence_test package can reach the
// concrete router while nothing here ships in a production binary.

// RouterForTest exposes the unexported router to the external test package.
type RouterForTest = router

// NewForTest supplies a real logger: SessionsWS logs on the write-failure path.
func NewForTest(pres model.Presence, del model.Delivery) *RouterForTest {
	return newRouter(pres, del, log.Default)
}
