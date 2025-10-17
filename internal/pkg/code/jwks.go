package code

import (
	"net/http"

	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// JWKS 密钥管理相关错误 (102xxx)
const (
	// ErrInvalidKid - 400: Invalid kid: kid cannot be empty.
	ErrInvalidKid = 102001

	// ErrInvalidJWK - 400: Invalid JWK: kty cannot be empty.
	ErrInvalidJWK = 102002

	// ErrInvalidJWKUse - 400: Invalid JWK: use must be 'sig'.
	ErrInvalidJWKUse = 102003

	// ErrInvalidJWKAlg - 400: Invalid JWK: alg cannot be empty.
	ErrInvalidJWKAlg = 102004

	// ErrKidMismatch - 400: Kid mismatch: key.Kid and JWK.Kid must be equal.
	ErrKidMismatch = 102005

	// ErrUnsupportedKty - 400: Unsupported key type.
	ErrUnsupportedKty = 102006

	// ErrMissingRSAParams - 400: Missing RSA parameters: n and e are required.
	ErrMissingRSAParams = 102007

	// ErrMissingECParams - 400: Missing EC parameters: crv, x, y are required.
	ErrMissingECParams = 102008

	// ErrMissingOKPParams - 400: Missing OKP parameters: crv, x are required.
	ErrMissingOKPParams = 102009

	// ErrInvalidStateTransition - 400: Invalid key state transition.
	ErrInvalidStateTransition = 102010

	// ErrInvalidTimeRange - 400: Invalid time range: NotAfter must be after NotBefore.
	ErrInvalidTimeRange = 102011

	// ErrEmptyJWKS - 400: JWKS cannot be empty.
	ErrEmptyJWKS = 102012

	// ErrInvalidRotationInterval - 400: Rotation interval must be positive.
	ErrInvalidRotationInterval = 102013

	// ErrInvalidGracePeriod - 400: Grace period must be positive.
	ErrInvalidGracePeriod = 102014

	// ErrInvalidMaxKeys - 400: Max keys must be at least 2.
	ErrInvalidMaxKeys = 102015

	// ErrGracePeriodTooLong - 400: Grace period must be shorter than rotation interval.
	ErrGracePeriodTooLong = 102016

	// ErrKeyNotFound - 404: Key not found.
	ErrKeyNotFound = 102020

	// ErrNoActiveKey - 404: No active key available.
	ErrNoActiveKey = 102021

	// ErrKeyAlreadyExists - 409: Key with this kid already exists.
	ErrKeyAlreadyExists = 102022
)

// nolint: gochecknoinits
func init() {
	register(ErrInvalidKid, http.StatusBadRequest, "Invalid kid: kid cannot be empty")
	register(ErrInvalidJWK, http.StatusBadRequest, "Invalid JWK: kty cannot be empty")
	register(ErrInvalidJWKUse, http.StatusBadRequest, "Invalid JWK: use must be 'sig'")
	register(ErrInvalidJWKAlg, http.StatusBadRequest, "Invalid JWK: alg cannot be empty")
	register(ErrKidMismatch, http.StatusBadRequest, "Kid mismatch: key.Kid and JWK.Kid must be equal")
	register(ErrUnsupportedKty, http.StatusBadRequest, "Unsupported key type")
	register(ErrMissingRSAParams, http.StatusBadRequest, "Missing RSA parameters: n and e are required")
	register(ErrMissingECParams, http.StatusBadRequest, "Missing EC parameters: crv, x, y are required")
	register(ErrMissingOKPParams, http.StatusBadRequest, "Missing OKP parameters: crv, x are required")
	register(ErrInvalidStateTransition, http.StatusBadRequest, "Invalid key state transition")
	register(ErrInvalidTimeRange, http.StatusBadRequest, "Invalid time range: NotAfter must be after NotBefore")
	register(ErrEmptyJWKS, http.StatusBadRequest, "JWKS cannot be empty")
	register(ErrInvalidRotationInterval, http.StatusBadRequest, "Rotation interval must be positive")
	register(ErrInvalidGracePeriod, http.StatusBadRequest, "Grace period must be positive")
	register(ErrInvalidMaxKeys, http.StatusBadRequest, "Max keys must be at least 2")
	register(ErrGracePeriodTooLong, http.StatusBadRequest, "Grace period must be shorter than rotation interval")
	register(ErrKeyNotFound, http.StatusNotFound, "Key not found")
	register(ErrNoActiveKey, http.StatusNotFound, "No active key available")
	register(ErrKeyAlreadyExists, http.StatusConflict, "Key with this kid already exists")
}

func register(code int, httpStatus int, message string) {
	errors.MustRegister(&jwksCoder{
		code:   code,
		status: httpStatus,
		msg:    message,
	})
}

// jwksCoder 实现 errors.Coder 接口
type jwksCoder struct {
	code   int
	status int
	msg    string
}

func (c *jwksCoder) Code() int {
	return c.code
}

func (c *jwksCoder) HTTPStatus() int {
	return c.status
}

func (c *jwksCoder) String() string {
	return c.msg
}

func (c *jwksCoder) Reference() string {
	return ""
}
