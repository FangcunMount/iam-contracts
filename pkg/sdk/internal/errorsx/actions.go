package errorsx

import "google.golang.org/grpc/codes"

type ErrorAction int

const (
	ActionNone ErrorAction = iota
	ActionRetry
	ActionReauth
	ActionForbidden
	ActionNotFound
	ActionBadRequest
	ActionRateLimit
	ActionFailover
	ActionEscalate
)

func (a ErrorAction) String() string {
	switch a {
	case ActionRetry:
		return "retry"
	case ActionReauth:
		return "reauth"
	case ActionForbidden:
		return "forbidden"
	case ActionNotFound:
		return "not_found"
	case ActionBadRequest:
		return "bad_request"
	case ActionRateLimit:
		return "rate_limit"
	case ActionFailover:
		return "failover"
	case ActionEscalate:
		return "escalate"
	default:
		return "none"
	}
}

func suggestAction(code codes.Code) ErrorAction {
	switch code {
	case codes.Unavailable, codes.Aborted:
		return ActionRetry
	case codes.DeadlineExceeded:
		return ActionFailover
	case codes.ResourceExhausted:
		return ActionRateLimit
	case codes.Unauthenticated:
		return ActionReauth
	case codes.PermissionDenied:
		return ActionForbidden
	case codes.NotFound:
		return ActionNotFound
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return ActionBadRequest
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return ActionEscalate
	default:
		return ActionNone
	}
}
