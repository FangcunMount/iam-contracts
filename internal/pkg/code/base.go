package code

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
