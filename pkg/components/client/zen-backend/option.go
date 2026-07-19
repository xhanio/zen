package zenbackend

import (
	"time"

	framingoClient "github.com/xhanio/framingo/pkg/services/api/client"
	"github.com/xhanio/framingo/pkg/utils/log"
)

type Option func(*client)

// WithTimeout sets the per-request timeout on the underlying HTTP client.
// Default: framingo's default (no timeout if unset).
func WithTimeout(d time.Duration) Option {
	return func(c *client) {
		c.fopts = append(c.fopts, framingoClient.WithTimeout(d))
	}
}

// WithLogger attaches a logger to the underlying framingo client; useful for
// observing request/response activity in tests or debugging.
func WithLogger(logger log.Logger) Option {
	return func(c *client) {
		c.fopts = append(c.fopts, framingoClient.WithLogger(logger))
	}
}

// WithDebug turns on framingo's per-request debug logging.
func WithDebug() Option {
	return func(c *client) {
		c.fopts = append(c.fopts, framingoClient.WithDebug())
	}
}

// WithAuthToken sets a bearer token attached as `Authorization: Bearer <token>`
// to every outbound request. Reserved for future use (zen v1 has no auth).
func WithAuthToken(token string) Option {
	return func(c *client) {
		c.token = token
	}
}
