package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Mock ChildRepository ====================

type MockChildRepository struct {
	mock.Mock
}

func (m *MockChildRepository) Create(ctx context.Context, child *domain.Child) error {
	args := m.Called(ctx, child)
	return args.Error(0)
}

func (m *MockChildRepository) FindByID(ctx context.Context, id domain.ChildID) (*domain.Child, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Child), args.Error(1)
}

func (m *MockChildRepository) FindByName(ctx context.Context, name string) (*domain.Child, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Child), args.Error(1)
}

func (m *MockChildRepository) FindByIDCard(ctx context.Context, idCard meta.IDCard) (*domain.Child, error) {
	args := m.Called(ctx, idCard)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Child), args.Error(1)
}

func (m *MockChildRepository) FindListByName(ctx context.Context, name string) ([]*domain.Child, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Child), args.Error(1)
}

func (m *MockChildRepository) FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*domain.Child, error) {
	args := m.Called(ctx, name, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Child), args.Error(1)
}

func (m *MockChildRepository) FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*domain.Child, error) {
	args := m.Called(ctx, name, gender, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Child), args.Error(1)
}

func (m *MockChildRepository) Update(ctx context.Context, child *domain.Child) error {
	args := m.Called(ctx, child)
	return args.Error(0)
}

// ==================== ChildRegister 测试 ====================

func TestNewRegisterService(t *testing.T) {
	mockRepo := new(MockChildRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	assert.NotNil(t, registerSvc)
}

func TestChildRegister_Register_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()

	name := "小明"
	gender := meta.GenderMale
	birthday := meta.NewBirthday("2015-06-01")

	// Act
	child, err := registerSvc.Register(ctx, name, gender, birthday)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, child)
	assert.Equal(t, name, child.Name)
	assert.Equal(t, gender, child.Gender)
	assert.Equal(t, birthday, child.Birthday)
	mockRepo.AssertExpectations(t)
}

func TestChildRegister_Register_EmptyName_ShouldFail(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()

	name := ""
	gender := meta.GenderMale
	birthday := meta.NewBirthday("2015-06-01")

	// Act
	child, err := registerSvc.Register(ctx, name, gender, birthday)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, child)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	mockRepo.AssertExpectations(t)
}

func TestChildRegister_RegisterWithIDCard_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	registerSvc := service.NewRegisterService(mockRepo)
	ctx := context.Background()

	name := "小红"
	gender := meta.GenderFemale
	birthday := meta.NewBirthday("2016-03-15")
	idCard := meta.NewIDCard("小红", "110101201603151234")

	// Act
	child, err := registerSvc.RegisterWithIDCard(ctx, name, gender, birthday, idCard)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, child)
	assert.Equal(t, name, child.Name)
	assert.Equal(t, gender, child.Gender)
	assert.Equal(t, birthday, child.Birthday)
	assert.Equal(t, idCard, child.IDCard)
	mockRepo.AssertExpectations(t)
}

// ==================== ChildQueryer 测试 ====================

func TestNewQueryService(t *testing.T) {
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	assert.NotNil(t, querySvc)
}

func TestChildQueryer_FindByID_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(12345)
	expectedChild, _ := domain.NewChild("小李")
	expectedChild.ID = childID

	mockRepo.On("FindByID", ctx, childID).Return(expectedChild, nil)

	// Act
	child, err := querySvc.FindByID(ctx, childID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, child)
	assert.Equal(t, childID, child.ID)
	assert.Equal(t, "小李", child.Name)
	mockRepo.AssertExpectations(t)
}

func TestChildQueryer_FindByID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(99999)
	mockRepo.On("FindByID", ctx, childID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	child, err := querySvc.FindByID(ctx, childID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, child)
	assert.True(t, errors.IsCode(err, code.ErrUserNotFound))
	mockRepo.AssertExpectations(t)
}

func TestChildQueryer_FindByIDCard_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()

	idCard := meta.NewIDCard("小王", "110101201701011234")
	expectedChild, _ := domain.NewChild("小王", domain.WithIDCard(idCard))

	mockRepo.On("FindByIDCard", ctx, idCard).Return(expectedChild, nil)

	// Act
	child, err := querySvc.FindByIDCard(ctx, idCard)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, child)
	assert.Equal(t, "小王", child.Name)
	assert.Equal(t, idCard, child.IDCard)
	mockRepo.AssertExpectations(t)
}

func TestChildQueryer_FindListByName_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()

	name := "小张"
	child1, _ := domain.NewChild(name)
	child1.ID = domain.NewChildID(1)
	child2, _ := domain.NewChild(name)
	child2.ID = domain.NewChildID(2)
	expectedChildren := []*domain.Child{child1, child2}

	mockRepo.On("FindListByName", ctx, name).Return(expectedChildren, nil)

	// Act
	children, err := querySvc.FindListByName(ctx, name)

	// Assert
	require.NoError(t, err)
	require.Len(t, children, 2)
	assert.Equal(t, name, children[0].Name)
	assert.Equal(t, name, children[1].Name)
	mockRepo.AssertExpectations(t)
}

func TestChildQueryer_FindSimilar_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	querySvc := service.NewQueryService(mockRepo)
	ctx := context.Background()

	name := "小赵"
	gender := meta.GenderMale
	birthday := meta.NewBirthday("2018-01-01")

	child1, _ := domain.NewChild(name, domain.WithGender(gender), domain.WithBirthday(birthday))
	child1.ID = domain.NewChildID(1)
	expectedChildren := []*domain.Child{child1}

	mockRepo.On("FindSimilar", ctx, name, gender, birthday).Return(expectedChildren, nil)

	// Act
	children, err := querySvc.FindSimilar(ctx, name, gender, birthday)

	// Assert
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, name, children[0].Name)
	assert.Equal(t, gender, children[0].Gender)
	mockRepo.AssertExpectations(t)
}

// ==================== ChildProfileEditor 测试 ====================

func TestNewProfileService(t *testing.T) {
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	assert.NotNil(t, profileSvc)
}

func TestChildProfileEditor_Rename_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(12345)
	oldChild, _ := domain.NewChild("旧名字")
	oldChild.ID = childID
	newName := "新名字"

	mockRepo.On("FindByID", ctx, childID).Return(oldChild, nil)

	// Act
	updatedChild, err := profileSvc.Rename(ctx, childID, newName)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedChild)
	assert.Equal(t, newName, updatedChild.Name)
	mockRepo.AssertExpectations(t)
}

func TestChildProfileEditor_UpdateIDCard_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(12345)
	child, _ := domain.NewChild("小孙")
	child.ID = childID
	newIDCard := meta.NewIDCard("小孙", "320106201801011234")

	mockRepo.On("FindByID", ctx, childID).Return(child, nil)

	// Act
	updatedChild, err := profileSvc.UpdateIDCard(ctx, childID, newIDCard)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedChild)
	assert.Equal(t, newIDCard, updatedChild.IDCard)
	mockRepo.AssertExpectations(t)
}

func TestChildProfileEditor_UpdateProfile_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(12345)
	child, _ := domain.NewChild("小周")
	child.ID = childID
	newGender := meta.GenderFemale
	newBirthday := meta.NewBirthday("2019-05-20")

	mockRepo.On("FindByID", ctx, childID).Return(child, nil)

	// Act
	updatedChild, err := profileSvc.UpdateProfile(ctx, childID, newGender, newBirthday)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedChild)
	assert.Equal(t, newGender, updatedChild.Gender)
	assert.Equal(t, newBirthday, updatedChild.Birthday)
	mockRepo.AssertExpectations(t)
}

func TestChildProfileEditor_UpdateHeightWeight_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(12345)
	child, _ := domain.NewChild("小吴")
	child.ID = childID
	newHeight, _ := meta.NewHeightFromFloat(115.5)
	newWeight, _ := meta.NewWeightFromFloat(22.3)

	mockRepo.On("FindByID", ctx, childID).Return(child, nil)

	// Act
	updatedChild, err := profileSvc.UpdateHeightWeight(ctx, childID, newHeight, newWeight)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, updatedChild)
	assert.Equal(t, newHeight, updatedChild.Height)
	assert.Equal(t, newWeight, updatedChild.Weight)
	mockRepo.AssertExpectations(t)
}

func TestChildProfileEditor_ChildNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockChildRepository)
	profileSvc := service.NewProfileService(mockRepo)
	ctx := context.Background()

	childID := domain.NewChildID(99999)
	mockRepo.On("FindByID", ctx, childID).Return(nil, gorm.ErrRecordNotFound)

	// Act
	updatedChild, err := profileSvc.Rename(ctx, childID, "新名字")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedChild)
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "not found")
	mockRepo.AssertExpectations(t)
}
