package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks/service"
)

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// MockKeyRepository 模拟密钥仓储
type MockKeyRepository struct {
	mock.Mock
}

func (m *MockKeyRepository) Save(ctx context.Context, key *jwks.Key) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockKeyRepository) Update(ctx context.Context, key *jwks.Key) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockKeyRepository) Delete(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyRepository) FindByKid(ctx context.Context, kid string) (*jwks.Key, error) {
	args := m.Called(ctx, kid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyRepository) FindByStatus(ctx context.Context, status jwks.KeyStatus) ([]*jwks.Key, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jwks.Key), args.Error(1)
}

func (m *MockKeyRepository) CountByStatus(ctx context.Context, status jwks.KeyStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockKeyRepository) List(ctx context.Context, offset, limit int) ([]*jwks.Key, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*jwks.Key), args.Get(1).(int64), args.Error(2)
}

func (m *MockKeyRepository) FindAll(ctx context.Context, limit, offset int) ([]*jwks.Key, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*jwks.Key), args.Get(1).(int64), args.Error(2)
}

func (m *MockKeyRepository) FindPublishable(ctx context.Context) ([]*jwks.Key, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jwks.Key), args.Error(1)
}

func (m *MockKeyRepository) FindExpired(ctx context.Context) ([]*jwks.Key, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jwks.Key), args.Error(1)
}

// MockKeyGenerator 模拟密钥生成器
type MockKeyGenerator struct {
	mock.Mock
}

func (m *MockKeyGenerator) GenerateKeyPair(ctx context.Context, alg, kid string) (*driven.KeyPair, error) {
	args := m.Called(ctx, alg, kid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*driven.KeyPair), args.Error(1)
}

func (m *MockKeyGenerator) SupportedAlgorithms() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// TestKeyRotation_RotateKey 测试密钥轮换
func TestKeyRotation_RotateKey(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockKeyRepository, *MockKeyGenerator)
		expectError   bool
		errorContains string
	}{
		{
			name: "成功轮换：无现有密钥",
			setupMocks: func(repo *MockKeyRepository, gen *MockKeyGenerator) {
				// 没有Active密钥
				repo.On("FindByStatus", mock.Anything, jwks.KeyActive).
					Return([]*jwks.Key{}, nil)

				// 生成新密钥
				newKeyPair := &driven.KeyPair{
					PublicJWK: jwks.PublicJWK{
						Kty: "RSA",
						Use: "sig",
						Alg: "RS256",
						Kid: "new-key",
						N:   strPtr("test-n"),
						E:   strPtr("AQAB"),
					},
					PrivateKey: "private-key",
				}
				gen.On("GenerateKeyPair", mock.Anything, "RS256", mock.AnythingOfType("string")).
					Return(newKeyPair, nil)

				// 保存新密钥
				repo.On("Save", mock.Anything, mock.AnythingOfType("*jwks.Key")).
					Return(nil)

				// 清理超过MaxKeys的密钥
				repo.On("CountByStatus", mock.Anything, jwks.KeyActive).Return(int64(1), nil)
				repo.On("CountByStatus", mock.Anything, jwks.KeyGrace).Return(int64(0), nil)

				// 清理过期Retired密钥
				repo.On("FindByStatus", mock.Anything, jwks.KeyRetired).
					Return([]*jwks.Key{}, nil)
			},
			expectError: false,
		},
		{
			name: "成功轮换：将现有Active密钥转为Grace",
			setupMocks: func(repo *MockKeyRepository, gen *MockKeyGenerator) {
				// 现有Active密钥
				now := time.Now()
				oldKey := jwks.NewKey("old-key", jwks.PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "old-key",
					N:   strPtr("old-n"),
					E:   strPtr("AQAB"),
				}, jwks.WithNotBefore(now.Add(-30*24*time.Hour)))

				repo.On("FindByStatus", mock.Anything, jwks.KeyActive).
					Return([]*jwks.Key{oldKey}, nil)

				// 更新旧密钥为Grace状态
				repo.On("Update", mock.Anything, mock.MatchedBy(func(k *jwks.Key) bool {
					return k.Kid == "old-key" && k.Status == jwks.KeyGrace
				})).Return(nil)

				// 生成新密钥
				newKeyPair := &driven.KeyPair{
					PublicJWK: jwks.PublicJWK{
						Kty: "RSA",
						Use: "sig",
						Alg: "RS256",
						Kid: "new-key",
						N:   strPtr("new-n"),
						E:   strPtr("AQAB"),
					},
					PrivateKey: "new-private-key",
				}
				gen.On("GenerateKeyPair", mock.Anything, "RS256", mock.AnythingOfType("string")).
					Return(newKeyPair, nil)

				// 保存新密钥
				repo.On("Save", mock.Anything, mock.AnythingOfType("*jwks.Key")).
					Return(nil)

				// 清理
				repo.On("CountByStatus", mock.Anything, jwks.KeyActive).Return(int64(1), nil)
				repo.On("CountByStatus", mock.Anything, jwks.KeyGrace).Return(int64(1), nil)
				repo.On("FindByStatus", mock.Anything, jwks.KeyRetired).Return([]*jwks.Key{}, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置mocks
			mockRepo := new(MockKeyRepository)
			mockGen := new(MockKeyGenerator)
			tt.setupMocks(mockRepo, mockGen)

			// 创建服务
			logger := log.New(log.NewOptions())
			policy := jwks.DefaultRotationPolicy()
			rotationSvc := service.NewKeyRotation(mockRepo, mockGen, policy, logger)

			// 执行轮换
			ctx := context.Background()
			newKey, err := rotationSvc.RotateKey(ctx)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, newKey)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, newKey)
				assert.Equal(t, jwks.KeyActive, newKey.Status)
			}

			// 验证所有mock调用
			mockRepo.AssertExpectations(t)
			mockGen.AssertExpectations(t)
		})
	}
}

// TestKeyRotation_ShouldRotate 测试是否需要轮换
func TestKeyRotation_ShouldRotate(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func(*MockKeyRepository)
		expectRotate bool
		expectError  bool
	}{
		{
			name: "无Active密钥：需要轮换",
			setupMocks: func(repo *MockKeyRepository) {
				repo.On("FindByStatus", mock.Anything, jwks.KeyActive).
					Return([]*jwks.Key{}, nil)
			},
			expectRotate: true,
			expectError:  false,
		},
		{
			name: "Active密钥未到轮换时间：不需要轮换",
			setupMocks: func(repo *MockKeyRepository) {
				now := time.Now()
				recentKey := jwks.NewKey("recent-key", jwks.PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "recent-key",
					N:   strPtr("test-n"),
					E:   strPtr("AQAB"),
				}, jwks.WithNotBefore(now.Add(-1*24*time.Hour))) // 1天前

				repo.On("FindByStatus", mock.Anything, jwks.KeyActive).
					Return([]*jwks.Key{recentKey}, nil)
			},
			expectRotate: false,
			expectError:  false,
		},
		{
			name: "Active密钥已到轮换时间：需要轮换",
			setupMocks: func(repo *MockKeyRepository) {
				now := time.Now()
				oldKey := jwks.NewKey("old-key", jwks.PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "old-key",
					N:   strPtr("test-n"),
					E:   strPtr("AQAB"),
				}, jwks.WithNotBefore(now.Add(-31*24*time.Hour))) // 31天前

				repo.On("FindByStatus", mock.Anything, jwks.KeyActive).
					Return([]*jwks.Key{oldKey}, nil)
			},
			expectRotate: true,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置mocks
			mockRepo := new(MockKeyRepository)
			mockGen := new(MockKeyGenerator)
			tt.setupMocks(mockRepo)

			// 创建服务
			logger := log.New(log.NewOptions())
			policy := jwks.DefaultRotationPolicy()
			rotationSvc := service.NewKeyRotation(mockRepo, mockGen, policy, logger)

			// 检查是否需要轮换
			ctx := context.Background()
			shouldRotate, err := rotationSvc.ShouldRotate(ctx)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRotate, shouldRotate)
			}

			// 验证所有mock调用
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestKeyRotation_GetRotationStatus 测试获取轮换状态
func TestKeyRotation_GetRotationStatus(t *testing.T) {
	mockRepo := new(MockKeyRepository)
	mockGen := new(MockKeyGenerator)

	// 设置mocks
	now := time.Now()
	activeKey := jwks.NewKey("active-key", jwks.PublicJWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: "active-key",
		N:   strPtr("test-n"),
		E:   strPtr("AQAB"),
	}, jwks.WithNotBefore(now.Add(-10*24*time.Hour)))

	graceKey := jwks.NewKey("grace-key", jwks.PublicJWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: "grace-key",
		N:   strPtr("test-n2"),
		E:   strPtr("AQAB"),
	}, jwks.WithNotBefore(now.Add(-40*24*time.Hour)))
	_ = graceKey.EnterGrace()

	mockRepo.On("FindByStatus", mock.Anything, jwks.KeyActive).
		Return([]*jwks.Key{activeKey}, nil)
	mockRepo.On("FindByStatus", mock.Anything, jwks.KeyGrace).
		Return([]*jwks.Key{graceKey}, nil)
	mockRepo.On("CountByStatus", mock.Anything, jwks.KeyRetired).
		Return(int64(5), nil)

	// 创建服务
	logger := log.New(log.NewOptions())
	policy := jwks.DefaultRotationPolicy()
	rotationSvc := service.NewKeyRotation(mockRepo, mockGen, policy, logger)

	// 获取轮换状态
	ctx := context.Background()
	status, err := rotationSvc.GetRotationStatus(ctx)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.NotNil(t, status.ActiveKey)
	assert.Equal(t, "active-key", status.ActiveKey.Kid)
	assert.Len(t, status.GraceKeys, 1)
	assert.Equal(t, "grace-key", status.GraceKeys[0].Kid)
	assert.Equal(t, 5, status.RetiredKeys)

	mockRepo.AssertExpectations(t)
}

// TestKeyRotation_UpdateRotationPolicy 测试更新轮换策略
func TestKeyRotation_UpdateRotationPolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        jwks.RotationPolicy
		expectError   bool
		errorContains string
	}{
		{
			name: "有效策略",
			policy: jwks.RotationPolicy{
				RotationInterval: 30 * 24 * time.Hour,
				GracePeriod:      7 * 24 * time.Hour,
				MaxKeysInJWKS:    3,
			},
			expectError: false,
		},
		{
			name: "无效策略：RotationInterval为0",
			policy: jwks.RotationPolicy{
				RotationInterval: 0,
				GracePeriod:      7 * 24 * time.Hour,
				MaxKeysInJWKS:    3,
			},
			expectError:   true,
			errorContains: "must be positive",
		},
		{
			name: "无效策略：GracePeriod >= RotationInterval",
			policy: jwks.RotationPolicy{
				RotationInterval: 7 * 24 * time.Hour,
				GracePeriod:      7 * 24 * time.Hour,
				MaxKeysInJWKS:    3,
			},
			expectError:   true,
			errorContains: "must be shorter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockKeyRepository)
			mockGen := new(MockKeyGenerator)

			logger := log.New(log.NewOptions())
			defaultPolicy := jwks.DefaultRotationPolicy()
			rotationSvc := service.NewKeyRotation(mockRepo, mockGen, defaultPolicy, logger)

			// 更新策略
			ctx := context.Background()
			err := rotationSvc.UpdateRotationPolicy(ctx, tt.policy)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// 验证策略已更新
				updatedPolicy := rotationSvc.GetRotationPolicy()
				assert.Equal(t, tt.policy.RotationInterval, updatedPolicy.RotationInterval)
				assert.Equal(t, tt.policy.GracePeriod, updatedPolicy.GracePeriod)
				assert.Equal(t, tt.policy.MaxKeysInJWKS, updatedPolicy.MaxKeysInJWKS)
			}
		})
	}
}
