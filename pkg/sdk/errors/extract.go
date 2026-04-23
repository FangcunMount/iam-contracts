package errors

import (
	stdErrors "errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GRPCCode(err error) codes.Code {
	var iamErr *IAMError
	if stdErrors.As(err, &iamErr) {
		return iamErr.GRPCCode
	}

	if st, ok := status.FromError(err); ok {
		return st.Code()
	}

	return codes.Unknown
}

func Message(err error) string {
	var iamErr *IAMError
	if stdErrors.As(err, &iamErr) {
		return iamErr.Message
	}

	if st, ok := status.FromError(err); ok {
		return st.Message()
	}

	if err != nil {
		return err.Error()
	}
	return ""
}

func AsIAMError(err error) (*IAMError, bool) {
	var iamErr *IAMError
	if stdErrors.As(err, &iamErr) {
		return iamErr, true
	}
	return nil, false
}
