package entities

import "time"

// Clock abstracts the current time so that services can be deterministically
// tested (ARCHITECTURE.md §6). Never call time.Now() directly in entities or
// services — inject a Clock instead.
type Clock interface {
	Now() time.Time
}
