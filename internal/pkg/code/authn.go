package code

// Authentication errors (1xxxxx).
const (
	// ErrTokenInvalid - 401: Token invalid.
	ErrTokenInvalid = 100005
)

// AuthN-specific errors (100201+).
const (
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

	// ErrTokenGeneration - 500: Failed to generate token.
	ErrTokenGeneration = 100208
)
