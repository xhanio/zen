package mcp_test

import (
	"context"
	"testing"

	"github.com/xhanio/zen/pkg/services/mcp"
)

func TestMCP_Init_ConstructsServer(t *testing.T) {
	svc := mcp.New(nil) // ping tool doesn't touch backend
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if svc.Handler() == nil {
		t.Fatal("Handler() returned nil after Init")
	}
}
