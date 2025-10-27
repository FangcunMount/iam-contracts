package code

import (
	"net/http"

	"github.com/FangcunMount/component-base/pkg/errors"
)

// Authentication errors (100201+).
const (
	// ErrTokenInvalid - 401: Token invalid.
	ErrTokenInvalid = 100005

	// ErrEncrypt - 401: Error occurred while encrypting the user password.
	ErrEncrypt = 100201

	// ErrSignatureInvalid - 401: Signature is invalid.
	ErrSignatureInvalid = 100202

	// ErrExpired - 401: Token expired.
	ErrExpired = 100203

	// ErrInvalidAuthHeader - 401: Invalid authorization header.
	ErrInvalidAuthHeader = 100204

	// ErrMissingHeader - 401: The `Authorization` header was empty.
	ErrMissingHeader = 100205

	// ErrPasswordIncorrect - 401: Password was incorrect.
	ErrPasswordIncorrect = 100206
)

// nolint: gochecknoinits
func init() {
	registerAuthn()
}

func registerAuthn() {
	errors.MustRegister(&authnCoder{code: ErrTokenInvalid, status: http.StatusUnauthorized, msg: "Token invalid"})
	errors.MustRegister(&authnCoder{code: ErrEncrypt, status: http.StatusUnauthorized, msg: "Error occurred while encrypting the user password"})
	errors.MustRegister(&authnCoder{code: ErrSignatureInvalid, status: http.StatusUnauthorized, msg: "Signature is invalid"})
	errors.MustRegister(&authnCoder{code: ErrExpired, status: http.StatusUnauthorized, msg: "Token expired"})
	errors.MustRegister(&authnCoder{code: ErrInvalidAuthHeader, status: http.StatusUnauthorized, msg: "Invalid authorization header"})
	errors.MustRegister(&authnCoder{code: ErrMissingHeader, status: http.StatusUnauthorized, msg: "The `Authorization` header was empty"})
	errors.MustRegister(&authnCoder{code: ErrPasswordIncorrect, status: http.StatusUnauthorized, msg: "Password was incorrect"})
}

// authnCoder 实现 errors.Coder 接口
type authnCoder struct {
	code   int
	status int
	msg    string
}

func (c *authnCoder) Code() int {
	return c.code
}

func (c *authnCoder) HTTPStatus() int {
	return c.status
}

func (c *authnCoder) String() string {
	return c.msg
}

func (c *authnCoder) Reference() string {
	return ""
}
