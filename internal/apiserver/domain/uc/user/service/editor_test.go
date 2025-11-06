package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== UserProfileEditor 测试 ====================

func TestNewProfileService(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)

	// Act
	profileSvc := service.NewProfileService(mockRepo)

	// Assert
	assert.NotNil(t, profileSvc)
}

func TestUserProfileEditor_Rename_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("旧名字", meta.NewPhone("13800138000"))
	existingUser.ID = userID
	newName := "新名字"

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := profileSvc.Rename(ctx, userID, newName)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newName, updatedUser.Name)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_Rename_EmptyName_ShouldFail(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	emptyName := "   " // 空白字符串

	// Act
	updatedUser, err := profileSvc.Rename(ctx, userID, emptyName)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "nickname cannot be empty")
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_Rename_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(99999)
	newName := "新名字"

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := profileSvc.Rename(ctx, userID, newName)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateContact_Success_BothPhoneAndEmail(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("张三", meta.NewPhone("13800138000"))
	existingUser.ID = userID

	newPhone := meta.NewPhone("13900139000")
	newEmail := meta.NewEmail("zhangsan@example.com")

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	// Mock: 新手机号唯一性检查通过
	mockRepo.On("FindByPhone", ctx, newPhone).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := profileSvc.UpdateContact(ctx, userID, newPhone, newEmail)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newPhone, updatedUser.Phone)
	assert.Equal(t, newEmail, updatedUser.Email)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateContact_OnlyPhone(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("李四", meta.NewPhone("13800138000"))
	existingUser.ID = userID

	newPhone := meta.NewPhone("13900139000")
	emptyEmail := meta.Email{} // 空邮箱

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	// Mock: 新手机号唯一性检查通过
	mockRepo.On("FindByPhone", ctx, newPhone).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := profileSvc.UpdateContact(ctx, userID, newPhone, emptyEmail)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newPhone, updatedUser.Phone)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateContact_OnlyEmail(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	oldPhone := meta.NewPhone("13800138000")
	existingUser, _ := domain.NewUser("王五", oldPhone)
	existingUser.ID = userID

	samePhone := oldPhone // 保持手机号不变
	newEmail := meta.NewEmail("wangwu@example.com")

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := profileSvc.UpdateContact(ctx, userID, samePhone, newEmail)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, samePhone, updatedUser.Phone)
	assert.Equal(t, newEmail, updatedUser.Email)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateContact_PhoneAlreadyExists(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("赵六", meta.NewPhone("13800138000"))
	existingUser.ID = userID

	newPhone := meta.NewPhone("13900139000")
	newEmail := meta.NewEmail("zhaoliu@example.com")

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	// Mock: 新手机号已被其他用户使用
	otherUser, _ := domain.NewUser("其他用户", newPhone)
	mockRepo.On("FindByPhone", ctx, newPhone).Return(otherUser, nil)

	// Act
	updatedUser, err := profileSvc.UpdateContact(ctx, userID, newPhone, newEmail)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	assert.True(t, errors.IsCode(err, code.ErrUserAlreadyExists))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "already exists")
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateIDCard_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(12345)
	existingUser, _ := domain.NewUser("孙七", meta.NewPhone("13800138000"))
	existingUser.ID = userID

	newIDCard := meta.NewIDCard("孙七", "110101199001011234")

	// Mock: 查找用户成功
	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	updatedUser, err := profileSvc.UpdateIDCard(ctx, userID, newIDCard)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newIDCard, updatedUser.IDCard)
	mockRepo.AssertExpectations(t)
}

func TestUserProfileEditor_UpdateIDCard_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	userID := domain.NewUserID(99999)
	newIDCard := meta.NewIDCard("周八", "320106198501010001")

	// Mock: 用户不存在
	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedUser, err := profileSvc.UpdateIDCard(ctx, userID, newIDCard)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedUser)
	mockRepo.AssertExpectations(t)
}
