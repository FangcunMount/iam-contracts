package operation

import (
	"context"
	"github.com/FangcunMount/reliable-messaging/transaction"
)

type referenceSinkKey struct{}

func WithReferenceSink(ctx context.Context, sink func(string)) (context.Context, error) {
	if ctx == nil || sink == nil {
		return nil, ErrInvalid
	}
	return context.WithValue(ctx, referenceSinkKey{}, sink), nil
}
func ReportReference(ctx context.Context, id string) {
	if ctx == nil || !ValidID(id) {
		return
	}
	if sink, ok := ctx.Value(referenceSinkKey{}).(func(string)); ok {
		sink(id)
	}
}

type OutcomeError struct {
	OperationID string
	Local       transaction.LocalOutcome
	Cause       error
}

func (e *OutcomeError) Error() string {
	if e.Local == transaction.LocalAborted {
		return "original IAM operation is aborted; inspect its reference before starting another operation"
	}
	return "IAM operation outcome requires authoritative status lookup"
}
func (e *OutcomeError) Unwrap() error              { return e.Cause }
func (e *OutcomeError) OperationReference() string { return e.OperationID }
