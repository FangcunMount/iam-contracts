package errors

import (
	stdErrors "errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound           = stdErrors.New("not found")
	ErrUnauthorized       = stdErrors.New("unauthorized")
	ErrPermissionDenied   = stdErrors.New("permission denied")
	ErrInvalidArgument    = stdErrors.New("invalid argument")
	ErrAlreadyExists      = stdErrors.New("already exists")
	ErrServiceUnavailable = stdErrors.New("service unavailable")
	ErrInternal           = stdErrors.New("internal error")
	ErrTokenExpired       = stdErrors.New("token expired")
	ErrTokenInvalid       = stdErrors.New("token invalid")
	ErrTokenRevoked       = stdErrors.New("token revoked")
	ErrRateLimited        = stdErrors.New("rate limited")
	ErrTimeout            = stdErrors.New("timeout")
)

// IAMError 是 SDK 统一错误包装类型。
type IAMError struct {
	Code     string
	Message  string
	GRPCCode codes.Code
	Cause    error
}

func (e *IAMError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("iam: %s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("iam: %s: %s", e.Code, e.Message)
}

func (e *IAMError) Unwrap() error {
	return e.Cause
}

func (e *IAMError) Is(target error) bool {
	switch e.GRPCCode {
	case codes.NotFound:
		return target == ErrNotFound
	case codes.Unauthenticated:
		return target == ErrUnauthorized
	case codes.PermissionDenied:
		return target == ErrPermissionDenied
	case codes.InvalidArgument:
		return target == ErrInvalidArgument
	case codes.AlreadyExists:
		return target == ErrAlreadyExists
	case codes.Unavailable:
		return target == ErrServiceUnavailable
	case codes.Internal:
		return target == ErrInternal
	case codes.ResourceExhausted:
		return target == ErrRateLimited
	case codes.DeadlineExceeded:
		return target == ErrTimeout
	}
	return false
}

// Wrap 将 gRPC 错误包装为 IAMError。
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return &IAMError{
			Code:     "UNKNOWN",
			Message:  err.Error(),
			GRPCCode: codes.Unknown,
			Cause:    err,
		}
	}

	return &IAMError{
		Code:     st.Code().String(),
		Message:  st.Message(),
		GRPCCode: st.Code(),
		Cause:    err,
	}
}

// WrapWithCode 使用自定义错误码包装错误。
func WrapWithCode(err error, code string, message string) error {
	grpcCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		grpcCode = st.Code()
	}

	return &IAMError{
		Code:     code,
		Message:  message,
		GRPCCode: grpcCode,
		Cause:    err,
	}
}
