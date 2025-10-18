package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/service"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ==================== UserQueryer 测试 ====================

func TestNewQueryService(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)

	// Act
	querySvc := service.NewQueryService(mockRepo)

	// Assert
	assert.NotNil(t, querySvc)
}

func TestUserQueryer_FindByID_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	userID := domain.NewUserID(12345)
	expectedUser, _ := domain.NewUser("张三", meta.NewPhone("13800138000"))
	expectedUser.ID = userID

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(expectedUser, nil)

	// Act
	user, err := querySvc.FindByID(ctx, userID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "张三", user.Name)
	mockRepo.AssertExpectations(t)
}

func TestUserQueryer_FindByID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	userID := domain.NewUserID(99999)

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	user, err := querySvc.FindByID(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrUserNotFound))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "not found")
	mockRepo.AssertExpectations(t)
}

func TestUserQueryer_FindByID_DatabaseError(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	userID := domain.NewUserID(12345)
	dbError := fmt.Errorf("database connection failed")

	// Mock: 数据库查询失败
	mockRepo.On("FindByID", ctx, userID).Return(nil, dbError)

	// Act
	user, err := querySvc.FindByID(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrDatabase))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "find user by id")
	mockRepo.AssertExpectations(t)
}

func TestUserQueryer_FindByPhone_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	phone := meta.NewPhone("13800138000")
	expectedUser, _ := domain.NewUser("李四", phone)
	expectedUser.ID = domain.NewUserID(12345)

	// Mock: 查找用户成功
	mockRepo.On("FindByPhone", ctx, phone).Return(expectedUser, nil)

	// Act
	user, err := querySvc.FindByPhone(ctx, phone)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, phone, user.Phone)
	assert.Equal(t, "李四", user.Name)
	mockRepo.AssertExpectations(t)
}

func TestUserQueryer_FindByPhone_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	phone := meta.NewPhone("13900139000")

	// Mock: 用户不存在
	mockRepo.On("FindByPhone", ctx, phone).Return(nil, gorm.ErrRecordNotFound)

	// Act
	user, err := querySvc.FindByPhone(ctx, phone)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrUserNotFound))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "not found")
	mockRepo.AssertExpectations(t)
}

func TestUserQueryer_FindByPhone_DatabaseError(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	phone := meta.NewPhone("13800138000")
	dbError := fmt.Errorf("database connection timeout")

	// Mock: 数据库查询失败
	mockRepo.On("FindByPhone", ctx, phone).Return(nil, dbError)

	// Act
	user, err := querySvc.FindByPhone(ctx, phone)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrDatabase))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "find user by phone")
	mockRepo.AssertExpectations(t)
}

// ==================== 综合查询场景测试 ====================

func TestUserQueryer_MultipleQueryMethods(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()
	
	userID := domain.NewUserID(12345)
	phone := meta.NewPhone("13800138000")
	user, _ := domain.NewUser("王五", phone)
	user.ID = userID

	// Mock: 通过 ID 和 Phone 都能找到同一个用户
	mockRepo.On("FindByID", ctx, userID).Return(user, nil)
	mockRepo.On("FindByPhone", ctx, phone).Return(user, nil)

	// Act - 通过 ID 查询
	userByID, err1 := querySvc.FindByID(ctx, userID)
	
	// Act - 通过 Phone 查询
	userByPhone, err2 := querySvc.FindByPhone(ctx, phone)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NotNil(t, userByID)
	require.NotNil(t, userByPhone)
	
	// 验证返回的是同一个用户
	assert.Equal(t, userByID.ID, userByPhone.ID)
	assert.Equal(t, userByID.Name, userByPhone.Name)
	assert.Equal(t, userByID.Phone, userByPhone.Phone)
	
	mockRepo.AssertExpectations(t)
}
