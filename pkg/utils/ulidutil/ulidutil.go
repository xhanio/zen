package ulidutil

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/xhanio/errors"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// New returns a fresh Crockford base32 ulid (26 chars), monotonic within a
// single millisecond tick.
func New() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// Parse returns an errors.BadRequest if s is not a valid 26-char Crockford
// base32 ulid. Uses the strict parser so non-base32 characters are rejected.
func Parse(s string) error {
	if _, err := ulid.ParseStrict(s); err != nil {
		return errors.BadRequest.Wrapf(err, "invalid ulid %q", s)
	}
	return nil
}
