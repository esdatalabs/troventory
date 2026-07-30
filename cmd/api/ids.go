package main

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// newID returns a fresh correlation/idempotency identifier.
func newID() string {
	id, err := uuid.NewV4()
	if err != nil {
		// crypto/rand exhausted — effectively never happens; fall back to
		// a timestamp so the request can still proceed.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return id.String()
}
