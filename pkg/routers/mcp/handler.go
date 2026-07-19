package mcp

import "github.com/xhanio/zen/pkg/types/api"

// MCP bridges Echo's routing to the SDK's StreamableHTTPHandler.
// The MCP SDK's handler does its own routing under /mcp; we just hand
// over the http.ResponseWriter / *http.Request pair.
func (r *router) MCP(c api.Context) error {
	r.svc.Handler().ServeHTTP(c.Response().Writer, c.Request())
	return nil
}
