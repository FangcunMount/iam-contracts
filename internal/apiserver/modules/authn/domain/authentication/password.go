package authentication

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHashAlgorithm 密码哈希算法
type PasswordHashAlgorithm string

const (
	// AlgorithmBcrypt Bcrypt 算法
	AlgorithmBcrypt PasswordHashAlgorithm = "bcrypt"
	// AlgorithmArgon2 Argon2 算法（预留）
	AlgorithmArgon2 PasswordHashAlgorithm = "argon2"
)

// PasswordHash 密码哈希值对象
type PasswordHash struct {
	Hash       string                // 哈希值
	Algorithm  PasswordHashAlgorithm // 算法
	Parameters map[string]string     // 算法参数（如 salt、cost 等）
}

// NewPasswordHash 创建密码哈希
func NewPasswordHash(hash string, algorithm PasswordHashAlgorithm, parameters map[string]string) *PasswordHash {
	if parameters == nil {
		parameters = make(map[string]string)
	}
	return &PasswordHash{
		Hash:       hash,
		Algorithm:  algorithm,
		Parameters: parameters,
	}
}

// NewBcryptPasswordHash 创建 Bcrypt 密码哈希
func NewBcryptPasswordHash(hash string) *PasswordHash {
	return &PasswordHash{
		Hash:       hash,
		Algorithm:  AlgorithmBcrypt,
		Parameters: make(map[string]string),
	}
}

// Verify 验证明文密码是否匹配哈希值
func (p *PasswordHash) Verify(plainPassword string) (bool, error) {
	switch p.Algorithm {
	case AlgorithmBcrypt:
		return p.verifyBcrypt(plainPassword)
	case AlgorithmArgon2:
		return false, fmt.Errorf("argon2 algorithm not implemented yet")
	default:
		return false, fmt.Errorf("unsupported password hash algorithm: %s", p.Algorithm)
	}
}

// verifyBcrypt 使用 Bcrypt 验证密码
func (p *PasswordHash) verifyBcrypt(plainPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(p.Hash), []byte(plainPassword))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("bcrypt comparison failed: %w", err)
	}
	return true, nil
}

// HashPassword 对明文密码进行哈希（工厂方法）
func HashPassword(plainPassword string, algorithm PasswordHashAlgorithm) (*PasswordHash, error) {
	switch algorithm {
	case AlgorithmBcrypt:
		return hashWithBcrypt(plainPassword)
	case AlgorithmArgon2:
		return nil, fmt.Errorf("argon2 algorithm not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported password hash algorithm: %s", algorithm)
	}
}

// hashWithBcrypt 使用 Bcrypt 对密码进行哈希
func hashWithBcrypt(plainPassword string) (*PasswordHash, error) {
	// 使用默认 cost (10)
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt hash generation failed: %w", err)
	}
	return NewBcryptPasswordHash(string(hash)), nil
}

// SecureCompare 安全比较两个字符串（防止时序攻击）
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
