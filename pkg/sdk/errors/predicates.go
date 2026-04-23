package errors

import (
	stdErrors "errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func IsNotFound(err error) bool {
	return stdErrors.Is(err, ErrNotFound) || hasGRPCCode(err, codes.NotFound)
}

func IsUnauthorized(err error) bool {
	return stdErrors.Is(err, ErrUnauthorized) || hasGRPCCode(err, codes.Unauthenticated)
}

func IsPermissionDenied(err error) bool {
	return stdErrors.Is(err, ErrPermissionDenied) || hasGRPCCode(err, codes.PermissionDenied)
}

func IsInvalidArgument(err error) bool {
	return stdErrors.Is(err, ErrInvalidArgument) || hasGRPCCode(err, codes.InvalidArgument)
}

func IsAlreadyExists(err error) bool {
	return stdErrors.Is(err, ErrAlreadyExists) || hasGRPCCode(err, codes.AlreadyExists)
}

func IsServiceUnavailable(err error) bool {
	return stdErrors.Is(err, ErrServiceUnavailable) || hasGRPCCode(err, codes.Unavailable)
}

func IsTimeout(err error) bool {
	return stdErrors.Is(err, ErrTimeout) || hasGRPCCode(err, codes.DeadlineExceeded)
}

func IsRateLimited(err error) bool {
	return stdErrors.Is(err, ErrRateLimited) || hasGRPCCode(err, codes.ResourceExhausted)
}

func IsInternal(err error) bool {
	return stdErrors.Is(err, ErrInternal) || hasGRPCCode(err, codes.Internal)
}

func IsTokenExpired(err error) bool {
	return stdErrors.Is(err, ErrTokenExpired)
}

func IsTokenInvalid(err error) bool {
	return stdErrors.Is(err, ErrTokenInvalid)
}

func IsCanceled(err error) bool {
	return hasGRPCCode(err, codes.Canceled)
}

func IsRetryable(err error) bool {
	return hasGRPCCode(err, codes.Unavailable) ||
		hasGRPCCode(err, codes.ResourceExhausted) ||
		hasGRPCCode(err, codes.Aborted) ||
		hasGRPCCode(err, codes.DeadlineExceeded)
}

func hasGRPCCode(err error, code codes.Code) bool {
	if st, ok := status.FromError(err); ok {
		return st.Code() == code
	}

	var iamErr *IAMError
	if stdErrors.As(err, &iamErr) {
		return iamErr.GRPCCode == code
	}

	return false
}
