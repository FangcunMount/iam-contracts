package user_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== User 实体创建测试 ====================

func TestNewUser_Success(t *testing.T) {
	// 准备测试数据
	name := "张三"
	phone := meta.NewPhone("13800138000")

	// 执行测试
	u, err := user.NewUser(name, phone)

	// 断言
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, name, u.Name)
	assert.Equal(t, phone, u.Phone)
	assert.True(t, u.Email.IsEmpty(), "email should be empty by default")
	assert.Equal(t, "", u.IDCard.Number(), "id card should be empty by default")
	assert.Equal(t, user.UserActive, u.Status, "status should be UserActive by default")
}

func TestNewUser_WithOptions(t *testing.T) {
	// 准备测试数据
	name := "李四"
	phone := meta.NewPhone("13900139000")
	email := meta.NewEmail("lisi@example.com")
	idCard := meta.NewIDCard("李四", "110101199001011234")
	userID := user.NewUserID(12345)

	// 执行测试 - 使用选项
	u, err := user.NewUser(
		name,
		phone,
		user.WithID(userID),
		user.WithEmail(email),
		user.WithIDCard(idCard),
		user.WithStatus(user.UserActive),
	)

	// 断言
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, name, u.Name)
	assert.Equal(t, phone, u.Phone)
	assert.Equal(t, email, u.Email)
	assert.Equal(t, idCard, u.IDCard)
	assert.Equal(t, userID, u.ID)
	assert.Equal(t, user.UserActive, u.Status)
}

func TestNewUser_EmptyName_ShouldFail(t *testing.T) {
	// 准备测试数据
	name := ""
	phone := meta.NewPhone("13800138000")

	// 执行测试
	u, err := user.NewUser(name, phone)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, u)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	// 使用 %+v 格式化以获取详细错误信息
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "name cannot be empty")
}

func TestNewUser_EmptyPhone_ShouldFail(t *testing.T) {
	// 准备测试数据
	name := "王五"
	phone := meta.Phone{} // 空手机号

	// 执行测试
	u, err := user.NewUser(name, phone)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, u)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	// 使用 %+v 格式化以获取详细错误信息
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "phone cannot be empty")
}

// ==================== User 状态管理测试 ====================

func TestUser_Activate(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	u.Deactivate() // 先设为非活跃

	// 执行测试
	u.Activate()

	// 断言
	assert.Equal(t, user.UserActive, u.Status)
	assert.True(t, u.IsUsable())
	assert.False(t, u.IsInactive())
	assert.False(t, u.IsBlocked())
}

func TestUser_Deactivate(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	u.Activate() // 先设为活跃

	// 执行测试
	u.Deactivate()

	// 断言
	assert.Equal(t, user.UserInactive, u.Status)
	assert.False(t, u.IsUsable())
	assert.True(t, u.IsInactive())
	assert.False(t, u.IsBlocked())
}

func TestUser_Block(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	u.Activate() // 先设为活跃

	// 执行测试
	u.Block()

	// 断言
	assert.Equal(t, user.UserBlocked, u.Status)
	assert.False(t, u.IsUsable())
	assert.False(t, u.IsInactive())
	assert.True(t, u.IsBlocked())
}

// ==================== User 信息更新测试 ====================

func TestUser_UpdatePhone(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	newPhone := meta.NewPhone("13900139000")

	// 执行测试
	u.UpdatePhone(newPhone)

	// 断言
	assert.Equal(t, newPhone, u.Phone)
}

func TestUser_UpdateEmail(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	newEmail := meta.NewEmail("newemail@example.com")

	// 执行测试
	u.UpdateEmail(newEmail)

	// 断言
	assert.Equal(t, newEmail, u.Email)
}

func TestUser_UpdateIDCard(t *testing.T) {
	// 准备测试数据
	u, _ := user.NewUser("测试用户", meta.NewPhone("13800138000"))
	newIDCard := meta.NewIDCard("测试用户", "110101199001011234")

	// 执行测试
	u.UpdateIDCard(newIDCard)

	// 断言
	assert.Equal(t, newIDCard, u.IDCard)
}

// ==================== UserStatus 值对象测试 ====================

func TestUserStatus_Value(t *testing.T) {
	tests := []struct {
		name   string
		status user.UserStatus
		want   uint8
	}{
		{"Active", user.UserActive, 1},
		{"Inactive", user.UserInactive, 2},
		{"Blocked", user.UserBlocked, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Value())
		})
	}
}

func TestUserStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status user.UserStatus
		want   string
	}{
		{"Active", user.UserActive, "active"},
		{"Inactive", user.UserInactive, "inactive"},
		{"Blocked", user.UserBlocked, "blocked"},
		{"Unknown", user.UserStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

// ==================== UserID 值对象测试 ====================

func TestNewUserID(t *testing.T) {
	// 测试创建 UserID
	id := user.NewUserID(12345)
	assert.Equal(t, uint64(12345), id.Uint64())
}
