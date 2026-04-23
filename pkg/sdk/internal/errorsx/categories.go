package errorsx

import "google.golang.org/grpc/codes"

type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
	CategoryClient
	CategoryServer
	CategoryNetwork
	CategoryAuth
	CategoryValidation
	CategoryRateLimit
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryClient:
		return "client"
	case CategoryServer:
		return "server"
	case CategoryNetwork:
		return "network"
	case CategoryAuth:
		return "auth"
	case CategoryValidation:
		return "validation"
	case CategoryRateLimit:
		return "rate_limit"
	default:
		return "unknown"
	}
}

func categorize(code codes.Code) ErrorCategory {
	switch code {
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition:
		return CategoryValidation
	case codes.Unauthenticated, codes.PermissionDenied:
		return CategoryAuth
	case codes.NotFound, codes.AlreadyExists:
		return CategoryClient
	case codes.ResourceExhausted:
		return CategoryRateLimit
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return CategoryNetwork
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unimplemented:
		return CategoryServer
	default:
		return CategoryUnknown
	}
}

func isClientError(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
		codes.OutOfRange:
		return true
	}
	return false
}

func isServerError(code codes.Code) bool {
	switch code {
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unimplemented:
		return true
	}
	return false
}

func isRetryableCode(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded:
		return true
	}
	return false
}
