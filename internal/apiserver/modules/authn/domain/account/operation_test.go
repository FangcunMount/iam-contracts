package account

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewOperationAccount 测试创建运营后台账号
func TestNewOperationAccount(t *testing.T) {
	// Arrange
	accountID := NewAccountID(123)
	username := "admin"
	algo := "bcrypt"

	// Act
	opAccount := NewOperationAccount(accountID, username, algo)

	// Assert
	assert.Equal(t, accountID, opAccount.AccountID)
	assert.Equal(t, username, opAccount.Username)
	assert.Equal(t, algo, opAccount.Algo)
	assert.Nil(t, opAccount.PasswordHash)
	assert.Nil(t, opAccount.Params)
	assert.Equal(t, 0, opAccount.FailedAttempts)
	assert.Nil(t, opAccount.LockedUntil)
}

// TestNewOperationAccount_WithOptions 测试创建运营后台账号（带选项）
func TestNewOperationAccount_WithOptions(t *testing.T) {
	// Arrange
	accountID := NewAccountID(456)
	username := "operator"
	algo := "argon2id"
	passwordHash := []byte("hashed-password-123")
	params := []byte(`{"memory":65536,"iterations":3,"parallelism":4}`)
	lockedUntil := time.Now().Add(30 * time.Minute)
	lastChanged := time.Now().Add(-90 * 24 * time.Hour)

	// Act
	opAccount := NewOperationAccount(
		accountID,
		username,
		algo,
		WithPasswordHash(passwordHash),
		WithParams(params),
		WithFailedAttempts(3),
		WithLockedUntil(&lockedUntil),
		WithLastChangedAt(lastChanged),
	)

	// Assert
	assert.Equal(t, accountID, opAccount.AccountID)
	assert.Equal(t, username, opAccount.Username)
	assert.Equal(t, algo, opAccount.Algo)
	assert.Equal(t, passwordHash, opAccount.PasswordHash)
	assert.Equal(t, params, opAccount.Params)
	assert.Equal(t, 3, opAccount.FailedAttempts)
	assert.NotNil(t, opAccount.LockedUntil)
	assert.Equal(t, lockedUntil, *opAccount.LockedUntil)
	assert.Equal(t, lastChanged, opAccount.LastChangedAt)
}

// TestOperationAccount_IsLocked 测试账号锁定检查
func TestOperationAccount_IsLocked(t *testing.T) {
	t.Run("未锁定", func(t *testing.T) {
		// Arrange
		opAccount := NewOperationAccount(NewAccountID(123), "admin", "bcrypt")

		// Assert
		assert.False(t, opAccount.IsLocked())
	})

	t.Run("锁定未过期", func(t *testing.T) {
		// Arrange
		futureTime := time.Now().Add(30 * time.Minute)
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLockedUntil(&futureTime),
		)

		// Assert
		assert.True(t, opAccount.IsLocked())
	})

	t.Run("锁定已过期", func(t *testing.T) {
		// Arrange
		pastTime := time.Now().Add(-30 * time.Minute)
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLockedUntil(&pastTime),
		)

		// Assert
		assert.False(t, opAccount.IsLocked())
	})
}

// TestOperationAccount_Lock 测试锁定账号
func TestOperationAccount_Lock(t *testing.T) {
	// Arrange
	opAccount := NewOperationAccount(NewAccountID(123), "admin", "bcrypt")
	assert.False(t, opAccount.IsLocked())

	// Act
	lockUntil := time.Now().Add(1 * time.Hour)
	opAccount.Lock(lockUntil)

	// Assert
	assert.True(t, opAccount.IsLocked())
	assert.NotNil(t, opAccount.LockedUntil)
	assert.Equal(t, lockUntil, *opAccount.LockedUntil)
}

// TestOperationAccount_Unlock 测试解锁账号
func TestOperationAccount_Unlock(t *testing.T) {
	// Arrange
	futureTime := time.Now().Add(30 * time.Minute)
	opAccount := NewOperationAccount(
		NewAccountID(123),
		"admin",
		"bcrypt",
		WithLockedUntil(&futureTime),
		WithFailedAttempts(5),
	)
	assert.True(t, opAccount.IsLocked())

	// Act
	opAccount.Unlock()

	// Assert
	assert.False(t, opAccount.IsLocked())
	assert.Nil(t, opAccount.LockedUntil)
	assert.Equal(t, 0, opAccount.FailedAttempts, "解锁应重置失败尝试次数")
}

// TestOperationAccount_IncrementFailedAttempts 测试增加失败尝试次数
func TestOperationAccount_IncrementFailedAttempts(t *testing.T) {
	// Arrange
	opAccount := NewOperationAccount(NewAccountID(123), "admin", "bcrypt")
	assert.Equal(t, 0, opAccount.FailedAttempts)

	// Act & Assert
	opAccount.IncrementFailedAttempts()
	assert.Equal(t, 1, opAccount.FailedAttempts)

	opAccount.IncrementFailedAttempts()
	assert.Equal(t, 2, opAccount.FailedAttempts)

	opAccount.IncrementFailedAttempts()
	assert.Equal(t, 3, opAccount.FailedAttempts)
}

// TestOperationAccount_ResetFailedAttempts 测试重置失败尝试次数
func TestOperationAccount_ResetFailedAttempts(t *testing.T) {
	// Arrange
	opAccount := NewOperationAccount(
		NewAccountID(123),
		"admin",
		"bcrypt",
		WithFailedAttempts(5),
	)
	assert.Equal(t, 5, opAccount.FailedAttempts)

	// Act
	opAccount.ResetFailedAttempts()

	// Assert
	assert.Equal(t, 0, opAccount.FailedAttempts)
}

// TestOperationAccount_ChangePassword 测试修改密码
func TestOperationAccount_ChangePassword(t *testing.T) {
	// Arrange
	oldHash := []byte("old-password-hash")
	futureTime := time.Now().Add(30 * time.Minute)
	opAccount := NewOperationAccount(
		NewAccountID(123),
		"admin",
		"bcrypt",
		WithPasswordHash(oldHash),
		WithFailedAttempts(3),
		WithLockedUntil(&futureTime),
	)
	assert.True(t, opAccount.IsLocked())

	// Act
	newHash := []byte("new-password-hash")
	changedAt := time.Now()
	opAccount.ChangePassword(newHash, changedAt)

	// Assert
	assert.Equal(t, newHash, opAccount.PasswordHash, "密码哈希应更新")
	assert.Equal(t, changedAt, opAccount.LastChangedAt, "修改时间应更新")
	assert.Equal(t, 0, opAccount.FailedAttempts, "失败尝试次数应重置")
	assert.False(t, opAccount.IsLocked(), "账号应解锁")
}

// TestOperationAccount_IsPasswordExpired 测试密码过期检查
func TestOperationAccount_IsPasswordExpired(t *testing.T) {
	t.Run("未设置过期时间", func(t *testing.T) {
		// Arrange
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLastChangedAt(time.Now().Add(-100*24*time.Hour)),
		)

		// Assert
		assert.False(t, opAccount.IsPasswordExpired(0), "过期时间为 0 时不应过期")
		assert.False(t, opAccount.IsPasswordExpired(-1), "过期时间为负数时不应过期")
	})

	t.Run("密码未过期", func(t *testing.T) {
		// Arrange
		lastChanged := time.Now().Add(-30 * 24 * time.Hour) // 30 天前
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLastChangedAt(lastChanged),
		)

		// Assert - 90 天过期
		assert.False(t, opAccount.IsPasswordExpired(90*24*time.Hour))
	})

	t.Run("密码已过期", func(t *testing.T) {
		// Arrange
		lastChanged := time.Now().Add(-100 * 24 * time.Hour) // 100 天前
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLastChangedAt(lastChanged),
		)

		// Assert - 90 天过期
		assert.True(t, opAccount.IsPasswordExpired(90*24*time.Hour))
	})

	t.Run("边界情况 - 恰好过期", func(t *testing.T) {
		// Arrange
		lastChanged := time.Now().Add(-90*24*time.Hour - time.Second) // 刚好超过 90 天
		opAccount := NewOperationAccount(
			NewAccountID(123),
			"admin",
			"bcrypt",
			WithLastChangedAt(lastChanged),
		)

		// Assert
		assert.True(t, opAccount.IsPasswordExpired(90*24*time.Hour))
	})
}
