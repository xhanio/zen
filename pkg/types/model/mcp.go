package model

import (
	"net/http"

	"github.com/xhanio/framingo/pkg/types/common"
)

type MCP interface {
	common.Service
	common.Initializable
	Handler() http.Handler
}
