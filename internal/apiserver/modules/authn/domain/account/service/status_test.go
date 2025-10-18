package service

import (
	"context"
	"errors"
	"testing"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockAccountRepo 模拟账号仓储
type MockAccountRepo struct {
	CreateFunc            func(ctx context.Context, a *domain.Account) error
	FindByIDFunc          func(ctx context.Context, id domain.AccountID) (*domain.Account, error)
	FindByRefFunc         func(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error)
	UpdateStatusFunc      func(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error
	UpdateUserIDFunc      func(ctx context.Context, id domain.AccountID, userID domain.UserID) error
	UpdateExternalRefFunc func(ctx context.Context, id domain.AccountID, externalID string, appID *string) error
}

func (m *MockAccountRepo) Create(ctx context.Context, a *domain.Account) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, a)
	}
	return nil
}

func (m *MockAccountRepo) FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockAccountRepo) UpdateStatus(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockAccountRepo) FindByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error) {
	if m.FindByRefFunc != nil {
		return m.FindByRefFunc(ctx, provider, externalID, appID)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockAccountRepo) UpdateUserID(ctx context.Context, id domain.AccountID, userID domain.UserID) error {
	if m.UpdateUserIDFunc != nil {
		return m.UpdateUserIDFunc(ctx, id, userID)
	}
	return nil
}

func (m *MockAccountRepo) UpdateExternalRef(ctx context.Context, id domain.AccountID, externalID string, appID *string) error {
	if m.UpdateExternalRefFunc != nil {
		return m.UpdateExternalRefFunc(ctx, id, externalID, appID)
	}
	return nil
}

// TestStatusService_DisableAccount_Success 测试成功禁用账号
func TestStatusService_DisableAccount_Success(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(123)
	existingAccount := &domain.Account{
		ID:     accountID,
		UserID: domain.UserID(456),
		Status: domain.StatusActive,
	}

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			assert.Equal(t, accountID, id)
			return existingAccount, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error {
			assert.Equal(t, accountID, id)
			assert.Equal(t, domain.StatusDisabled, status)
			return nil
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.DisableAccount(ctx, accountID)

	// Assert
	assert.NoError(t, err)
}

// TestStatusService_DisableAccount_NotFound 测试禁用不存在的账号
func TestStatusService_DisableAccount_NotFound(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(999)

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.DisableAccount(ctx, accountID)

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestStatusService_DisableAccount_DatabaseError 测试数据库错误
func TestStatusService_DisableAccount_DatabaseError(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(123)

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			return nil, errors.New("database connection failed")
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.DisableAccount(ctx, accountID)

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrDatabase))
}

// TestStatusService_DisableAccount_UpdateFailed 测试更新状态失败
func TestStatusService_DisableAccount_UpdateFailed(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(123)
	existingAccount := &domain.Account{
		ID:     accountID,
		Status: domain.StatusActive,
	}

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			return existingAccount, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error {
			return errors.New("update failed")
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.DisableAccount(ctx, accountID)

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrDatabase))
}

// TestStatusService_EnableAccount_Success 测试成功启用账号
func TestStatusService_EnableAccount_Success(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(789)
	existingAccount := &domain.Account{
		ID:     accountID,
		UserID: domain.UserID(456),
		Status: domain.StatusDisabled,
	}

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			assert.Equal(t, accountID, id)
			return existingAccount, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error {
			assert.Equal(t, accountID, id)
			assert.Equal(t, domain.StatusActive, status)
			return nil
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.EnableAccount(ctx, accountID)

	// Assert
	assert.NoError(t, err)
}

// TestStatusService_EnableAccount_NotFound 测试启用不存在的账号
func TestStatusService_EnableAccount_NotFound(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(999)

	mockRepo := &MockAccountRepo{
		FindByIDFunc: func(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	service := NewStatusService(mockRepo)
	ctx := context.Background()

	// Act
	err := service.EnableAccount(ctx, accountID)

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}
