package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// ==================== Mock UserRepository ====================

// MockUserRepository 是 UserRepository 的 mock 实现
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByPhone(ctx context.Context, phone meta.Phone) (*domain.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// ==================== UserRegister 测试 ====================

func TestNewRegisterService(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)

	// Act
	registerSvc := service.NewRegisterService(mockRepo)

	// Assert
	assert.NotNil(t, registerSvc)
}

func TestUserRegister_Register_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()
	name := "张三"
	phone := meta.NewPhone("13800138000")

	// Mock: 手机号不存在（唯一性检查通过）
	mockRepo.On("FindByPhone", ctx, phone).Return(nil, gorm.ErrRecordNotFound)

	// Act
	user, err := registerSvc.Register(ctx, name, phone)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, name, user.Name)
	assert.Equal(t, phone, user.Phone)
	mockRepo.AssertExpectations(t)
}

func TestUserRegister_Register_PhoneAlreadyExists(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()
	name := "李四"
	phone := meta.NewPhone("13900139000")

	// Mock: 手机号已存在（返回一个已存在的用户）
	existingUser, _ := domain.NewUser("已存在用户", phone)
	mockRepo.On("FindByPhone", ctx, phone).Return(existingUser, nil)

	// Act
	user, err := registerSvc.Register(ctx, name, phone)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrUserAlreadyExists))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "already exists")
	mockRepo.AssertExpectations(t)
}

func TestUserRegister_Register_EmptyName_ShouldFail(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()
	name := "" // 空姓名
	phone := meta.NewPhone("13800138000")

	// Mock: 手机号不存在（唯一性检查通过）
	mockRepo.On("FindByPhone", ctx, phone).Return(nil, gorm.ErrRecordNotFound)

	// Act
	user, err := registerSvc.Register(ctx, name, phone)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "name cannot be empty")
	mockRepo.AssertExpectations(t)
}

func TestUserRegister_Register_DatabaseError(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()
	name := "王五"
	phone := meta.NewPhone("13700137000")

	// Mock: 数据库查询失败（非 RecordNotFound 错误）
	dbError := fmt.Errorf("database connection failed")
	mockRepo.On("FindByPhone", ctx, phone).Return(nil, dbError)

	// Act
	user, err := registerSvc.Register(ctx, name, phone)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, errors.IsCode(err, code.ErrDatabase))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "check user phone")
	mockRepo.AssertExpectations(t)
}
