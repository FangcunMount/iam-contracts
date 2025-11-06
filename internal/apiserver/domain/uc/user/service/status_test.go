package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== UserStatusChanger 测试 ====================

func TestNewStatusService(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)

	// Act
	statusSvc := service.NewStatusService(mockRepo)

	// Assert
	assert.NotNil(t, statusSvc)
}

func TestUserStatusChanger_Activate_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("张三", meta.NewPhone("13800138000"))
	existingUser.ID = userID
	existingUser.Deactivate() // 初始状态为停用

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := statusSvc.Activate(ctx, userID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, domain.UserActive, updatedUser.Status)
	mockRepo.AssertExpectations(t)
}

func TestUserStatusChanger_Activate_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(99999)

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := statusSvc.Activate(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	mockRepo.AssertExpectations(t)
}

func TestUserStatusChanger_Deactivate_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("李四", meta.NewPhone("13900139000"))
	existingUser.ID = userID
	existingUser.Activate() // 初始状态为激活

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := statusSvc.Deactivate(ctx, userID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, domain.UserInactive, updatedUser.Status)
	mockRepo.AssertExpectations(t)
}

func TestUserStatusChanger_Deactivate_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(99999)

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := statusSvc.Deactivate(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	mockRepo.AssertExpectations(t)
}

func TestUserStatusChanger_Block_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("王五", meta.NewPhone("13700137000"))
	existingUser.ID = userID
	existingUser.Activate() // 初始状态为激活

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := statusSvc.Block(ctx, userID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, domain.UserBlocked, updatedUser.Status)
	mockRepo.AssertExpectations(t)
}

func TestUserStatusChanger_Block_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(99999)

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := statusSvc.Block(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	mockRepo.AssertExpectations(t)
}

// ==================== 状态转换场景测试 ====================

func TestUserStatusChanger_StateTransitions(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	statusSvc := service.NewStatusService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)

	tests := []struct {
		name           string
		initialStatus  domain.UserStatus
		action         string
		expectedStatus domain.UserStatus
	}{
		{
			name:           "从激活到停用",
			initialStatus:  domain.UserActive,
			action:         "deactivate",
			expectedStatus: domain.UserInactive,
		},
		{
			name:           "从停用到激活",
			initialStatus:  domain.UserInactive,
			action:         "activate",
			expectedStatus: domain.UserActive,
		},
		{
			name:           "从激活到封禁",
			initialStatus:  domain.UserActive,
			action:         "block",
			expectedStatus: domain.UserBlocked,
		},
		{
			name:           "从封禁到激活",
			initialStatus:  domain.UserBlocked,
			action:         "activate",
			expectedStatus: domain.UserActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建用户并设置初始状态
			user, _ := domain.NewUser("测试用户", meta.NewPhone("13800138000"))
			user.ID = userID
			user.Status = tt.initialStatus

			// Mock: 查找用户成功
			mockRepo.On("FindByID", ctx, userID).Return(user, nil).Once()

			// Act
			var updatedUser *domain.User
			var err error
			switch tt.action {
			case "activate":
				updatedUser, err = statusSvc.Activate(ctx, userID)
			case "deactivate":
				updatedUser, err = statusSvc.Deactivate(ctx, userID)
			case "block":
				updatedUser, err = statusSvc.Block(ctx, userID)
			}

			// Assert
			require.NoError(t, err)
			require.NotNil(t, updatedUser)
			assert.Equal(t, tt.expectedStatus, updatedUser.Status)
		})
	}

	mockRepo.AssertExpectations(t)
}
