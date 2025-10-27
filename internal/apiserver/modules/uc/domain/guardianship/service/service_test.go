package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	childDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	guardport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/port"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship/service"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// ==================== Mock Repositories ====================

type MockGuardianshipRepository struct {
	mock.Mock
}

func (m *MockGuardianshipRepository) Create(ctx context.Context, guardianship *domain.Guardianship) error {
	args := m.Called(ctx, guardianship)
	return args.Error(0)
}

func (m *MockGuardianshipRepository) FindByID(ctx context.Context, id idutil.ID) (*domain.Guardianship, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Guardianship), args.Error(1)
}

func (m *MockGuardianshipRepository) FindByChildID(ctx context.Context, id childDomain.ChildID) ([]*domain.Guardianship, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Guardianship), args.Error(1)
}

func (m *MockGuardianshipRepository) FindByUserID(ctx context.Context, id userDomain.UserID) ([]*domain.Guardianship, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Guardianship), args.Error(1)
}

func (m *MockGuardianshipRepository) Update(ctx context.Context, guardianship *domain.Guardianship) error {
	args := m.Called(ctx, guardianship)
	return args.Error(0)
}

type MockChildRepository struct {
	mock.Mock
}

func (m *MockChildRepository) Create(ctx context.Context, child *childDomain.Child) error {
	args := m.Called(ctx, child)
	return args.Error(0)
}

func (m *MockChildRepository) FindByID(ctx context.Context, id childDomain.ChildID) (*childDomain.Child, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) FindByName(ctx context.Context, name string) (*childDomain.Child, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) FindByIDCard(ctx context.Context, idCard meta.IDCard) (*childDomain.Child, error) {
	args := m.Called(ctx, idCard)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) FindListByName(ctx context.Context, name string) ([]*childDomain.Child, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*childDomain.Child, error) {
	args := m.Called(ctx, name, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*childDomain.Child, error) {
	args := m.Called(ctx, name, gender, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*childDomain.Child), args.Error(1)
}

func (m *MockChildRepository) Update(ctx context.Context, child *childDomain.Child) error {
	args := m.Called(ctx, child)
	return args.Error(0)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *userDomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id userDomain.UserID) (*userDomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserRepository) FindByPhone(ctx context.Context, phone meta.Phone) (*userDomain.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userDomain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *userDomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// ==================== GuardianshipRegister 测试 ====================

func TestNewRegisterService(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockUserRepo)
	assert.NotNil(t, registerSvc)
}

func TestGuardianshipRegister_RegisterChildWithGuardian_Success(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	existingUser, _ := userDomain.NewUser("张三", meta.NewPhone("13800138000"))
	existingUser.ID = userID

	params := guardport.RegisterChildWithGuardianParams{
		UserID:   userID,
		Name:     "小明",
		Gender:   meta.GenderMale,
		Birthday: meta.NewBirthday("2015-05-01"),
		Relation: domain.RelParent,
	}

	mockUserRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	// Act
	guard, child, err := registerSvc.RegisterChildWithGuardian(ctx, params)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, guard)
	require.NotNil(t, child)
	assert.Equal(t, userID, guard.User)
	assert.Equal(t, domain.RelParent, guard.Rel)
	assert.Equal(t, "小明", child.Name)
	assert.Equal(t, meta.GenderMale, child.Gender)
	mockUserRepo.AssertExpectations(t)
}

func TestGuardianshipRegister_UserNotFound(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	registerSvc := service.NewRegisterService(mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(999)
	params := guardport.RegisterChildWithGuardianParams{
		UserID:   userID,
		Name:     "小红",
		Gender:   meta.GenderFemale,
		Birthday: meta.NewBirthday("2016-03-01"),
		Relation: domain.RelParent,
	}

	mockUserRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	guard, child, err := registerSvc.RegisterChildWithGuardian(ctx, params)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, guard)
	assert.Nil(t, child)
	mockUserRepo.AssertExpectations(t)
}

// ==================== GuardianshipManager 测试 ====================

func TestNewManagerService(t *testing.T) {
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	mockUserRepo := new(MockUserRepository)
	managerSvc := service.NewManagerService(mockGuardRepo, mockChildRepo, mockUserRepo)
	assert.NotNil(t, managerSvc)
}

func TestGuardianshipManager_AddGuardian_Success(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	mockUserRepo := new(MockUserRepository)
	managerSvc := service.NewManagerService(mockGuardRepo, mockChildRepo, mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childID := childDomain.NewChildID(200)

	user, _ := userDomain.NewUser("李四", meta.NewPhone("13900139000"))
	user.ID = userID

	child, _ := childDomain.NewChild("小李")
	child.ID = childID

	mockChildRepo.On("FindByID", ctx, childID).Return(child, nil)
	mockUserRepo.On("FindByID", ctx, userID).Return(user, nil)
	mockGuardRepo.On("FindByChildID", ctx, childID).Return([]*domain.Guardianship{}, nil)

	// Act
	guard, err := managerSvc.AddGuardian(ctx, userID, childID, domain.RelGrandparents)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, guard)
	assert.Equal(t, userID, guard.User)
	assert.Equal(t, childID, guard.Child)
	assert.Equal(t, domain.RelGrandparents, guard.Rel)
	assert.True(t, guard.IsActive())
	mockChildRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockGuardRepo.AssertExpectations(t)
}

func TestGuardianshipManager_AddGuardian_DuplicateGuardian(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	mockUserRepo := new(MockUserRepository)
	managerSvc := service.NewManagerService(mockGuardRepo, mockChildRepo, mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childID := childDomain.NewChildID(200)

	user, _ := userDomain.NewUser("王五", meta.NewPhone("13700137000"))
	user.ID = userID

	child, _ := childDomain.NewChild("小王")
	child.ID = childID

	// 已存在的监护关系
	existingGuard := &domain.Guardianship{
		User:          userID,
		Child:         childID,
		Rel:           domain.RelParent,
		EstablishedAt: time.Now().Add(-24 * time.Hour),
		RevokedAt:     nil,
	}

	mockChildRepo.On("FindByID", ctx, childID).Return(child, nil)
	mockUserRepo.On("FindByID", ctx, userID).Return(user, nil)
	mockGuardRepo.On("FindByChildID", ctx, childID).Return([]*domain.Guardianship{existingGuard}, nil)

	// Act
	guard, err := managerSvc.AddGuardian(ctx, userID, childID, domain.RelParent)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, guard)
	assert.True(t, errors.IsCode(err, code.ErrUserInvalid))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "already exists")
	mockChildRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockGuardRepo.AssertExpectations(t)
}

func TestGuardianshipManager_RemoveGuardian_Success(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	mockUserRepo := new(MockUserRepository)
	managerSvc := service.NewManagerService(mockGuardRepo, mockChildRepo, mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childID := childDomain.NewChildID(200)

	existingGuard := &domain.Guardianship{
		ID:            1,
		User:          userID,
		Child:         childID,
		Rel:           domain.RelParent,
		EstablishedAt: time.Now().Add(-30 * 24 * time.Hour),
		RevokedAt:     nil,
	}

	mockGuardRepo.On("FindByChildID", ctx, childID).Return([]*domain.Guardianship{existingGuard}, nil)

	// Act
	guard, err := managerSvc.RemoveGuardian(ctx, userID, childID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, guard)
	assert.False(t, guard.IsActive())
	assert.NotNil(t, guard.RevokedAt)
	mockGuardRepo.AssertExpectations(t)
}

func TestGuardianshipManager_RemoveGuardian_NotFound(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	mockUserRepo := new(MockUserRepository)
	managerSvc := service.NewManagerService(mockGuardRepo, mockChildRepo, mockUserRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childID := childDomain.NewChildID(200)

	mockGuardRepo.On("FindByChildID", ctx, childID).Return([]*domain.Guardianship{}, nil)

	// Act
	guard, err := managerSvc.RemoveGuardian(ctx, userID, childID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, guard)
	assert.True(t, errors.IsCode(err, code.ErrUserInvalid))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "not found")
	mockGuardRepo.AssertExpectations(t)
}

// ==================== GuardianshipQueryer 测试 ====================

func TestNewQueryService(t *testing.T) {
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockGuardRepo, mockChildRepo)
	assert.NotNil(t, querySvc)
}

func TestGuardianshipQueryer_FindByUserIDAndChildID_Success(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockGuardRepo, mockChildRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childID := childDomain.NewChildID(200)

	expectedGuard := &domain.Guardianship{
		ID:            1,
		User:          userID,
		Child:         childID,
		Rel:           domain.RelParent,
		EstablishedAt: time.Now(),
		RevokedAt:     nil,
	}

	mockGuardRepo.On("FindByChildID", ctx, childID).Return([]*domain.Guardianship{expectedGuard}, nil)

	// Act
	guard, err := querySvc.FindByUserIDAndChildID(ctx, userID, childID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, guard)
	assert.Equal(t, userID, guard.User)
	assert.Equal(t, childID, guard.Child)
	mockGuardRepo.AssertExpectations(t)
}

func TestGuardianshipQueryer_FindByUserIDAndChildName_Success(t *testing.T) {
	// Arrange
	mockGuardRepo := new(MockGuardianshipRepository)
	mockChildRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockGuardRepo, mockChildRepo)
	ctx := context.Background()

	userID := userDomain.NewUserID(100)
	childName := "小赵"

	child1, _ := childDomain.NewChild(childName)
	child1.ID = childDomain.NewChildID(201)
	child2, _ := childDomain.NewChild(childName)
	child2.ID = childDomain.NewChildID(202)

	guard1 := &domain.Guardianship{
		ID:            1,
		User:          userID,
		Child:         child1.ID,
		Rel:           domain.RelParent,
		EstablishedAt: time.Now(),
	}

	mockChildRepo.On("FindListByName", ctx, childName).Return([]*childDomain.Child{child1, child2}, nil)
	mockGuardRepo.On("FindByChildID", ctx, child1.ID).Return([]*domain.Guardianship{guard1}, nil)
	mockGuardRepo.On("FindByChildID", ctx, child2.ID).Return([]*domain.Guardianship{}, nil)

	// Act
	guards, err := querySvc.FindByUserIDAndChildName(ctx, userID, childName)

	// Assert
	require.NoError(t, err)
	require.Len(t, guards, 1)
	assert.Equal(t, userID, guards[0].User)
	assert.Equal(t, child1.ID, guards[0].Child)
	mockChildRepo.AssertExpectations(t)
	mockGuardRepo.AssertExpectations(t)
}
