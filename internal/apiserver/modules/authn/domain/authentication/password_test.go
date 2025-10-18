package authentication

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Password 值对象测试 ====================

func TestHashPassword_Bcrypt_Success(t *testing.T) {
	// Arrange
	plainPassword := "SecureP@ssw0rd123"

	// Act
	passwordHash, err := HashPassword(plainPassword, AlgorithmBcrypt)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, passwordHash)
	assert.Equal(t, AlgorithmBcrypt, passwordHash.Algorithm)
	assert.NotEmpty(t, passwordHash.Hash)
	assert.True(t, strings.HasPrefix(passwordHash.Hash, "$2a$")) // bcrypt 特征
}

func TestHashPassword_UnsupportedAlgorithm(t *testing.T) {
	// Arrange
	plainPassword := "password123"

	// Act
	passwordHash, err := HashPassword(plainPassword, "md5")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, passwordHash)
	assert.Contains(t, err.Error(), "unsupported password hash algorithm")
}

func TestHashPassword_Argon2_NotImplemented(t *testing.T) {
	// Arrange
	plainPassword := "password123"

	// Act
	passwordHash, err := HashPassword(plainPassword, AlgorithmArgon2)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, passwordHash)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestPasswordHash_Verify_Success(t *testing.T) {
	// Arrange
	plainPassword := "MySecretPassword!123"
	passwordHash, err := HashPassword(plainPassword, AlgorithmBcrypt)
	require.NoError(t, err)

	// Act
	match, err := passwordHash.Verify(plainPassword)

	// Assert
	require.NoError(t, err)
	assert.True(t, match)
}

func TestPasswordHash_Verify_WrongPassword(t *testing.T) {
	// Arrange
	correctPassword := "CorrectPassword123"
	wrongPassword := "WrongPassword456"
	passwordHash, err := HashPassword(correctPassword, AlgorithmBcrypt)
	require.NoError(t, err)

	// Act
	match, err := passwordHash.Verify(wrongPassword)

	// Assert
	require.NoError(t, err)
	assert.False(t, match, "Wrong password should not match")
}

func TestPasswordHash_Verify_EmptyPassword(t *testing.T) {
	// Arrange
	plainPassword := "password123"
	passwordHash, err := HashPassword(plainPassword, AlgorithmBcrypt)
	require.NoError(t, err)

	// Act
	match, err := passwordHash.Verify("")

	// Assert
	require.NoError(t, err)
	assert.False(t, match, "Empty password should not match")
}

func TestNewPasswordHash_Success(t *testing.T) {
	// Arrange
	hash := "$2a$10$abcdefghijklmnopqrstuv"
	algorithm := AlgorithmBcrypt
	parameters := map[string]string{"cost": "10"}

	// Act
	passwordHash := NewPasswordHash(hash, algorithm, parameters)

	// Assert
	require.NotNil(t, passwordHash)
	assert.Equal(t, hash, passwordHash.Hash)
	assert.Equal(t, algorithm, passwordHash.Algorithm)
	assert.Equal(t, "10", passwordHash.Parameters["cost"])
}

func TestNewPasswordHash_WithNilParameters(t *testing.T) {
	// Arrange
	hash := "$2a$10$xyz"
	algorithm := AlgorithmBcrypt

	// Act
	passwordHash := NewPasswordHash(hash, algorithm, nil)

	// Assert
	require.NotNil(t, passwordHash)
	assert.NotNil(t, passwordHash.Parameters)
	assert.Empty(t, passwordHash.Parameters)
}

func TestNewBcryptPasswordHash_Success(t *testing.T) {
	// Arrange
	hash := "$2a$10$N9qo8uLOickgx2ZMRZoMye"

	// Act
	passwordHash := NewBcryptPasswordHash(hash)

	// Assert
	require.NotNil(t, passwordHash)
	assert.Equal(t, hash, passwordHash.Hash)
	assert.Equal(t, AlgorithmBcrypt, passwordHash.Algorithm)
	assert.NotNil(t, passwordHash.Parameters)
}

func TestSecureCompare_Equal(t *testing.T) {
	// Arrange
	a := "secret_string_123"
	b := "secret_string_123"

	// Act
	result := SecureCompare(a, b)

	// Assert
	assert.True(t, result)
}

func TestSecureCompare_NotEqual(t *testing.T) {
	// Arrange
	a := "secret_string_123"
	b := "different_string"

	// Act
	result := SecureCompare(a, b)

	// Assert
	assert.False(t, result)
}

func TestSecureCompare_EmptyStrings(t *testing.T) {
	// Arrange
	a := ""
	b := ""

	// Act
	result := SecureCompare(a, b)

	// Assert
	assert.True(t, result)
}

// ==================== 边界测试 ====================

func TestPasswordHash_Verify_UnsupportedAlgorithm(t *testing.T) {
	// Arrange
	passwordHash := &PasswordHash{
		Hash:       "some_hash",
		Algorithm:  "unknown_algo",
		Parameters: make(map[string]string),
	}

	// Act
	match, err := passwordHash.Verify("password")

	// Assert
	assert.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "unsupported password hash algorithm")
}

func TestPasswordHash_DifferentPasswordsSameLength(t *testing.T) {
	// Arrange
	password1 := "abcdefgh"
	password2 := "12345678"
	passwordHash, err := HashPassword(password1, AlgorithmBcrypt)
	require.NoError(t, err)

	// Act
	match, err := passwordHash.Verify(password2)

	// Assert
	require.NoError(t, err)
	assert.False(t, match, "Different passwords of same length should not match")
}
