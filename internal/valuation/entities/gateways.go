package entities

import "context"

// AuditGateway records the outcome of a command. Because a Service's Send
// method has no return value, this is how any caller learns whether a
// command succeeded or failed.
type AuditGateway interface {
	RecordResult(ctx context.Context, result Result) error
}
