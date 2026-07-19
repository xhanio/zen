package mcp

import (
	"context"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (m *manager) Init(ctx context.Context) error {
	m.server = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "zen-mcp",
		Title:   "Zen",
		Version: "v0.1.0",
	}, nil)

	m.registerTools()

	m.handler = mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return m.server },
		nil,
	)
	return nil
}
