package guardianship_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
)

// ==================== Guardianship 实体测试 ====================

func TestGuardianship_IsActive_WhenNotRevoked(t *testing.T) {
	// Arrange
	g := &guardianship.Guardianship{
		ID:            1,
		User:          user.NewUserID(100),
		Child:         child.NewChildID(200),
		Rel:           guardianship.RelParent,
		EstablishedAt: time.Now(),
		RevokedAt:     nil,
	}

	// Act & Assert
	assert.True(t, g.IsActive())
}

func TestGuardianship_IsActive_WhenRevoked(t *testing.T) {
	// Arrange
	now := time.Now()
	g := &guardianship.Guardianship{
		ID:            1,
		User:          user.NewUserID(100),
		Child:         child.NewChildID(200),
		Rel:           guardianship.RelParent,
		EstablishedAt: time.Now().Add(-24 * time.Hour),
		RevokedAt:     &now,
	}

	// Act & Assert
	assert.False(t, g.IsActive())
}

func TestGuardianship_Revoke(t *testing.T) {
	// Arrange
	g := &guardianship.Guardianship{
		ID:            1,
		User:          user.NewUserID(100),
		Child:         child.NewChildID(200),
		Rel:           guardianship.RelGrandparents,
		EstablishedAt: time.Now().Add(-30 * 24 * time.Hour),
		RevokedAt:     nil,
	}

	assert.True(t, g.IsActive(), "监护关系应该是有效的")

	// Act
	revokeTime := time.Now()
	g.Revoke(revokeTime)

	// Assert
	assert.False(t, g.IsActive(), "监护关系应该已被撤销")
	assert.NotNil(t, g.RevokedAt)
	assert.Equal(t, revokeTime, *g.RevokedAt)
}

// ==================== Relation 常量测试 ====================

func TestRelationConstants(t *testing.T) {
	tests := []struct {
		name     string
		relation guardianship.Relation
		expected string
	}{
		{
			name:     "自己",
			relation: guardianship.RelSelf,
			expected: "self",
		},
		{
			name:     "父母",
			relation: guardianship.RelParent,
			expected: "parent",
		},
		{
			name:     "祖父母",
			relation: guardianship.RelGrandparents,
			expected: "grandparents",
		},
		{
			name:     "其他",
			relation: guardianship.RelOther,
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.relation))
		})
	}
}

// ==================== 综合场景测试 ====================

func TestGuardianship_CompleteLifecycle(t *testing.T) {
	// 创建监护关系
	userID := user.NewUserID(12345)
	childID := child.NewChildID(67890)
	establishedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	g := &guardianship.Guardianship{
		ID:            1,
		User:          userID,
		Child:         childID,
		Rel:           guardianship.RelParent,
		EstablishedAt: establishedAt,
		RevokedAt:     nil,
	}

	// 验证初始状态
	assert.Equal(t, int64(1), g.ID)
	assert.Equal(t, userID, g.User)
	assert.Equal(t, childID, g.Child)
	assert.Equal(t, guardianship.RelParent, g.Rel)
	assert.Equal(t, establishedAt, g.EstablishedAt)
	assert.True(t, g.IsActive())

	// 撤销监护关系
	revokeTime := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	g.Revoke(revokeTime)

	// 验证撤销后状态
	assert.False(t, g.IsActive())
	assert.NotNil(t, g.RevokedAt)
	assert.Equal(t, revokeTime, *g.RevokedAt)
}

func TestGuardianship_DifferentRelations(t *testing.T) {
	// 测试不同的监护关系类型
	relations := []guardianship.Relation{
		guardianship.RelSelf,
		guardianship.RelParent,
		guardianship.RelGrandparents,
		guardianship.RelOther,
	}

	for i, rel := range relations {
		t.Run(string(rel), func(t *testing.T) {
			g := &guardianship.Guardianship{
				ID:            int64(i + 1),
				User:          user.NewUserID(uint64(100 + i)),
				Child:         child.NewChildID(uint64(200 + i)),
				Rel:           rel,
				EstablishedAt: time.Now(),
				RevokedAt:     nil,
			}

			assert.Equal(t, rel, g.Rel)
			assert.True(t, g.IsActive())
		})
	}
}
