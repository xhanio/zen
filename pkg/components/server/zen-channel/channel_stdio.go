package zenchannel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/xhanio/framingo/pkg/utils/log"
)

// channelInstructions is returned in the initialize result and lands in the
// client's context automatically.
//
// It deliberately describes the event tag and then defers to the
// zen-conversation-watcher skill rather than restating the response rules. It
// used to summarize them, and the summary rotted: it still named document.update
// and "anchor card/document" long after v0.6 dropped documents entirely
// (migration 010_v06_card_only). Worse, a summary that reads as complete removes
// any reason to load the skill, so the reader never reaches the parts only the
// skill has — the link format, group anchors, how to fetch anchor content.
//
// Keep this to facts the skill cannot know (the wire shape) plus a pointer. The
// rules live in SKILL.md, once.
const channelInstructions = `Zen conversation events arrive as <channel source="zen" conversation_id="..." message_id="..." anchor_kind="..." anchor_id="..." has_selection="..."> blocks. anchor_kind is "card" or "group", and both anchor attributes are absent for a standalone conversation. When has_selection is true, the user's selected text is rendered as a quoted block before the message body. Reply with the reply tool — pass conversation_id back. Follow the zen-conversation-watcher skill for how to respond: it carries the decision tree, the reply-after-every-mutation rule, and the anchor-bounded action scope.`

// StdioServer is a minimal MCP server speaking newline-delimited JSON-RPC over
// In/Out. Hand-rolled because the github.com/modelcontextprotocol/go-sdk
// (v1.6.1) does not expose an API for sending arbitrary notification methods
// — notifications/claude/channel is experimental and unknown to the SDK's
// sendingMethodHandler, and the underlying jsonrpc2.Connection is unexported.
// The protocol surface we need (initialize / ping / tools/list / tools/call /
// notifications/initialized + one custom notification we push) is small
// enough to roll by hand.
type StdioServer struct {
	In    io.Reader
	Out   io.Writer
	Reply ReplyFunc

	ServerName    string // defaults to "zen-channel"
	ServerVersion string // defaults to "0.3.1"

	Log log.Logger

	writeMu sync.Mutex

	// readyCh closes when the MCP client completes `initialize`. Until then the
	// channel must not dial the backend or write notifications: stdout is the
	// JSON-RPC pipe, and a frame written before the handshake is a protocol
	// violation the client will not understand.
	//
	// readyOnce guards the allocation (Ready may be called before Run);
	// closeOnce guards the close (a client may send initialize twice).
	readyOnce sync.Once
	closeOnce sync.Once
	readyCh   chan struct{}

	clientMu   sync.RWMutex
	clientName string
	clientVer  string
}

// Ready closes once the MCP client has sent `initialize`.
func (s *StdioServer) Ready() <-chan struct{} {
	s.initReady()
	return s.readyCh
}

// ClientInfo returns the peer's name and version, captured from the
// `initialize` params. Empty strings before the handshake.
func (s *StdioServer) ClientInfo() (string, string) {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.clientName, s.clientVer
}

func (s *StdioServer) initReady() {
	s.readyOnce.Do(func() { s.readyCh = make(chan struct{}) })
}

type jsonrpcEnvelope struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Run drives the frame loop. Returns when stdin closes or ctx is cancelled.
func (s *StdioServer) Run(ctx context.Context) error {
	if s.ServerName == "" {
		s.ServerName = "zen-channel"
	}
	if s.ServerVersion == "" {
		s.ServerVersion = "0.3.1"
	}
	if s.Log == nil {
		s.Log = log.New(log.NoStdout())
	}
	s.initReady()

	scanner := bufio.NewScanner(s.In)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var env jsonrpcEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			s.Log.Warnf("bad frame: %v", err)
			continue
		}
		s.dispatch(ctx, &env)
	}
	return scanner.Err()
}

func (s *StdioServer) dispatch(ctx context.Context, env *jsonrpcEnvelope) {
	s.Log.Debugf("recv method=%q id=%s params=%s", env.Method, string(env.ID), string(env.Params))
	switch {
	case env.Method == "initialize":
		var params struct {
			ClientInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		// Params are optional; an absent clientInfo just leaves the fields empty.
		_ = json.Unmarshal(env.Params, &params)
		s.clientMu.Lock()
		s.clientName = params.ClientInfo.Name
		s.clientVer = params.ClientInfo.Version
		s.clientMu.Unlock()

		s.respond(env.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":        map[string]any{},
				"experimental": map[string]any{"claude/channel": map[string]any{}},
			},
			"serverInfo":   map[string]any{"name": s.ServerName, "version": s.ServerVersion},
			"instructions": channelInstructions,
		})

		// Signal after responding: the client is listening from here on.
		s.initReady()
		s.closeOnce.Do(func() { close(s.readyCh) })
	case env.Method == "ping":
		s.respond(env.ID, map[string]any{})
	case env.Method == "tools/list":
		s.respond(env.ID, map[string]any{
			"tools": []map[string]any{{
				"name":        "reply",
				"description": "Append an assistant message to a Zen conversation. Pass the conversation_id from the inbound channel event.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"conversation_id": map[string]any{"type": "string"},
						"content":         map[string]any{"type": "string"},
					},
					"required": []string{"conversation_id", "content"},
				},
			}},
		})
	case env.Method == "tools/call":
		s.handleToolsCall(ctx, env)
	case strings.HasPrefix(env.Method, "notifications/"):
		// No response. Includes notifications/initialized and any cancels.
	default:
		if env.ID != nil {
			s.respondErr(env.ID, -32601, "method not found: "+env.Method)
		}
	}
}

func (s *StdioServer) handleToolsCall(ctx context.Context, env *jsonrpcEnvelope) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	_ = json.Unmarshal(env.Params, &params)
	if params.Name != "reply" {
		s.respond(env.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": "unknown tool: " + params.Name}},
			"isError": true,
		})
		return
	}
	convID, _ := params.Arguments["conversation_id"].(string)
	content, _ := params.Arguments["content"].(string)
	out, err := s.Reply(ctx, convID, content)
	if err != nil {
		s.respond(env.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": "reply: " + err.Error()}},
			"isError": true,
		})
		return
	}
	s.respond(env.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": out}},
	})
}

// PushChannelNotification emits a notifications/claude/channel envelope on Out.
// Safe to call from any goroutine — the write is mutex-guarded against frame
// interleaving with the response path.
func (s *StdioServer) PushChannelNotification(_ context.Context, note ChannelNotification) error {
	return s.writeJSON(jsonrpcEnvelope{
		Jsonrpc: "2.0",
		Method:  "notifications/claude/channel",
		Params:  mustMarshalRaw(note),
	})
}

func (s *StdioServer) respond(id json.RawMessage, result any) {
	if err := s.writeJSON(jsonrpcEnvelope{Jsonrpc: "2.0", ID: id, Result: result}); err != nil {
		s.Log.Errorf("write response: %v", err)
	}
}

func (s *StdioServer) respondErr(id json.RawMessage, code int, msg string) {
	if err := s.writeJSON(jsonrpcEnvelope{Jsonrpc: "2.0", ID: id, Error: &jsonrpcError{Code: code, Message: msg}}); err != nil {
		s.Log.Errorf("write error response: %v", err)
	}
}

func (s *StdioServer) writeJSON(env jsonrpcEnvelope) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	s.Log.Debugf("send %s", string(b))
	b = append(b, '\n')
	_, err = s.Out.Write(b)
	return err
}

func mustMarshalRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("marshal: %v", err))
	}
	return b
}
