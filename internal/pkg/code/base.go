package code

import (
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Common: basic errors (1xxxxx).
const (
	// ErrSuccess - 200: OK.
	ErrSuccess = 100001

	// ErrUnknown - 500: Internal server error.
	ErrUnknown = 100002

	// ErrBind - 400: Error occurred while binding the request body to the struct.
	ErrBind = 100003

	// ErrValidation - 400: Validation failed.
	ErrValidation = 100004

	// ErrPageNotFound - 404: Page not found.
	ErrPageNotFound = 100006

	// ErrInvalidArgument - 400: Invalid argument.
	ErrInvalidArgument = 100007

	// ErrInvalidMessage - 400: Invalid message.
	ErrInvalidMessage = 100008
)

// common: database errors.
const (
	// ErrDatabase - 500: Database error.
	ErrDatabase int = iota + 100101
)

// common: encode/decode errors.
const (
	// ErrEncodingFailed - 500: Encoding failed due to an error with the data.
	ErrEncodingFailed int = iota + 100301

	// ErrDecodingFailed - 500: Decoding failed due to an error with the data.
	ErrDecodingFailed

	// ErrInvalidJSON - 500: Data is not valid JSON.
	ErrInvalidJSON

	// ErrEncodingJSON - 500: JSON data could not be encoded.
	ErrEncodingJSON

	// ErrDecodingJSON - 500: JSON data could not be decoded.
	ErrDecodingJSON

	// ErrInvalidYaml - 500: Data is not valid Yaml.
	ErrInvalidYaml

	// ErrEncodingYaml - 500: Yaml data could not be encoded.
	ErrEncodingYaml

	// ErrDecodingYaml - 500: Yaml data could not be decoded.
	ErrDecodingYaml
)

// common: module errors.
const (
	// ErrModuleInitializationFailed - 500: Module initialization failed.
	ErrModuleInitializationFailed int = iota + 100401

	// ErrModuleNotFound - 404: Module not found.
	ErrModuleNotFound
)

// Common: internal server failure.
const (
	// ErrInternalServerError - 500: Internal server error.
	ErrInternalServerError = 100209
)

// Common: authentication and authorization errors.
const (
	// ErrUnauthenticated - 401: Authentication failed.
	ErrUnauthenticated int = iota + 100501

	// ErrUnauthorized - 403: Authorization failed.
	ErrUnauthorized

	// ErrInvalidCredentials - 401: Invalid credentials.
	ErrInvalidCredentials
)

func init() {
	registerBase(ErrSuccess, 200, "OK")
	registerBase(ErrUnknown, 500, "Internal server error")
	registerBase(ErrBind, 400, "Error occurred while binding the request body to the struct")
	registerBase(ErrValidation, 400, "Validation failed")
	registerBase(ErrPageNotFound, 404, "Page not found")
	registerBase(ErrInvalidArgument, 400, "Invalid argument")
	registerBase(ErrInvalidMessage, 400, "Invalid message")
	registerBase(ErrDatabase, 500, "Database error")
	registerBase(ErrEncodingFailed, 500, "Encoding failed due to an error with the data")
	registerBase(ErrDecodingFailed, 500, "Decoding failed due to an error with the data")
	registerBase(ErrInvalidJSON, 500, "Data is not valid JSON")
	registerBase(ErrEncodingJSON, 500, "JSON data could not be encoded")
	registerBase(ErrDecodingJSON, 500, "JSON data could not be decoded")
	registerBase(ErrInvalidYaml, 500, "Data is not valid Yaml")
	registerBase(ErrEncodingYaml, 500, "Yaml data could not be encoded")
	registerBase(ErrDecodingYaml, 500, "Yaml data could not be decoded")
	registerBase(ErrModuleInitializationFailed, 500, "Module initialization failed")
	registerBase(ErrModuleNotFound, 404, "Module not found")
	registerBase(ErrInternalServerError, 500, "Internal server error")
	registerBase(ErrUnauthenticated, 401, "Authentication failed")
	registerBase(ErrUnauthorized, 403, "Authorization failed")
	registerBase(ErrInvalidCredentials, 401, "Invalid credentials")
}

func registerBase(code int, httpStatus int, message string) {
	errors.MustRegister(&baseCoder{
		code:       code,
		httpStatus: httpStatus,
		message:    message,
	})
}

type baseCoder struct {
	code       int
	httpStatus int
	message    string
}

func (c *baseCoder) Code() int {
	return c.code
}

func (c *baseCoder) String() string {
	return c.message
}

func (c *baseCoder) Reference() string {
	return ""
}

func (c *baseCoder) HTTPStatus() int {
	return c.httpStatus
}
