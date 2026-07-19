package entity

import "time"

// Channel is one live zen-mcp channel process — one StreamWS connection.
//
// InstanceID is the connection identity: fresh per process, used for logs and
// to tell a reconnecting channel from the stale socket it replaces. SessionID
// is the routing identity: stable across a channel restart, because Claude Code
// re-spawns its MCP server within the same session.
type Channel struct {
	InstanceID  string    `json:"instance_id"`
	SessionID   string    `json:"session_id"`
	Cwd         string    `json:"cwd"`
	StartedAt   time.Time `json:"started_at"`
	ClientName  string    `json:"client_name"`
	ClientVer   string    `json:"client_version"`
	ConnectedAt time.Time `json:"connected_at"`
}

// ChannelRegistration is the first frame a channel writes on the /_stream/ws
// socket. The backend refuses to register a connection that has not sent one.
type ChannelRegistration struct {
	Kind       string    `json:"kind"`
	InstanceID string    `json:"instance_id"`
	SessionID  string    `json:"session_id"`
	Cwd        string    `json:"cwd"`
	StartedAt  time.Time `json:"started_at"`
	ClientName string    `json:"client_name"`
	ClientVer  string    `json:"client_version"`
}

// ChannelRegistrationKind is the Kind value on a ChannelRegistration frame.
const ChannelRegistrationKind = "register"

// ChannelDisplacedReason is the WebSocket close reason the backend sends to a
// channel whose SessionID has been claimed by a newer InstanceID.
//
// The displaced channel must treat this as terminal and stop, not reconnect.
// Two live processes sharing a SessionID would otherwise evict each other in a
// loop, each dialing back and reclaiming the session. That is reachable in
// practice: every process spawned from a Claude Code session inherits
// CLAUDE_CODE_SESSION_ID, so a channel started from a shell inside a session
// collides with that session's own channel.
const ChannelDisplacedReason = "displaced by a newer channel"
