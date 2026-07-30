// Package clock provides a single real-time clock implementation shared by
// every feature. Each feature declares its own entities.Clock interface
// (ARCHITECTURE.md §6), but since every one of them has the identical
// Now() time.Time method set, one concrete type satisfies all of them
// structurally — no per-feature duplication needed at the Provider layer.
package clock

import "time"

// Real is a Clock backed by the actual wall-clock time, always in UTC.
type Real struct{}

// Now returns the current time in UTC.
func (Real) Now() time.Time {
	return time.Now().UTC()
}
