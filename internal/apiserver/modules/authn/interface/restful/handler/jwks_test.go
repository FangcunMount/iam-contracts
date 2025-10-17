package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	jwksApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driving"
	"github.com/fangcun-mount/iam-contracts/pkg/log"
)

// MockKeyManagementService 模拟领域层密钥管理服务
type MockKeyManagementService struct {
	mock.Mock
}

func (m *MockKeyManagementService) CreateKey(ctx context.Context, alg string, notBefore, notAfter *time.Time) (*jwks.Key, error) {
	args := m.Called(ctx, alg, notBefore, notAfter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) GetActiveKey(ctx context.Context) (*jwks.Key, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) GetKeyByKid(ctx context.Context, kid string) (*jwks.Key, error) {
	args := m.Called(ctx, kid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) RetireKey(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) ForceRetireKey(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) EnterGracePeriod(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) CleanupExpiredKeys(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockKeyManagementService) ListKeys(ctx context.Context, status jwks.KeyStatus, limit, offset int) ([]*jwks.Key, int64, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*jwks.Key), args.Get(1).(int64), args.Error(2)
}

// MockKeySetPublishService 模拟领域层密钥集发布服务
type MockKeySetPublishService struct {
	mock.Mock
}

func (m *MockKeySetPublishService) BuildJWKS(ctx context.Context) ([]byte, driving.CacheTag, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, driving.CacheTag{}, args.Error(2)
	}
	return args.Get(0).([]byte), args.Get(1).(driving.CacheTag), args.Error(2)
}

func (m *MockKeySetPublishService) GetPublishableKeys(ctx context.Context) ([]*jwks.Key, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*jwks.Key), args.Error(1)
}

func (m *MockKeySetPublishService) ValidateCacheTag(ctx context.Context, clientTag driving.CacheTag) (bool, error) {
	args := m.Called(ctx, clientTag)
	return args.Bool(0), args.Error(1)
}

func (m *MockKeySetPublishService) GetCurrentCacheTag(ctx context.Context) (driving.CacheTag, error) {
	args := m.Called(ctx)
	return args.Get(0).(driving.CacheTag), args.Error(1)
}

func (m *MockKeySetPublishService) RefreshCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// setupTestHandler 设置测试处理器（使用真实的应用服务 + mock 的领域服务）
func setupTestHandler() (*JWKSHandler, *MockKeyManagementService, *MockKeySetPublishService) {
	mockKeyMgmtSvc := new(MockKeyManagementService)
	mockKeyPublishSvc := new(MockKeySetPublishService)

	// 直接使用全局 logger
	logger := log.WithName("test")

	// 创建真实的应用服务，注入 mock 的领域服务
	keyMgmtApp := jwksApp.NewKeyManagementAppService(mockKeyMgmtSvc, logger)
	keyPublishApp := jwksApp.NewKeyPublishAppService(mockKeyPublishSvc, logger)

	handler := NewJWKSHandler(keyMgmtApp, keyPublishApp)
	return handler, mockKeyMgmtSvc, mockKeyPublishSvc
} // setupGinTest 设置 Gin 测试环境
func setupGinTest() {
	gin.SetMode(gin.TestMode)
	log.Init(log.NewOptions())
}

func init() {
	setupGinTest()
}

// createTestKey 创建测试 Key 实体
func createTestKey(kid, algorithm string, status jwks.KeyStatus) *jwks.Key {
	now := time.Now()
	n := "test-modulus"
	e := "AQAB"
	publicJWK := jwks.PublicJWK{
		Kty: "RSA",
		Use: "sig",
		Alg: algorithm,
		Kid: kid,
		N:   &n,
		E:   &e,
	}

	key := &jwks.Key{
		Kid:       kid,
		Status:    status,
		JWK:       publicJWK,
		NotBefore: &now,
		NotAfter:  nil,
	}
	return key
}

// TestGetJWKS 测试获取 JWKS
func TestGetJWKS(t *testing.T) {
	setupGinTest()

	t.Run("成功获取 JWKS", func(t *testing.T) {
		handler, _, mockPublish := setupTestHandler()

		jwksJSON := []byte(`{"keys":[{"kty":"RSA","kid":"test-key-1"}]}`)
		lastModified := time.Now().UTC()
		etag := "test-etag-123"

		mockPublish.On("BuildJWKS", mock.Anything).Return(
			jwksJSON,
			driving.CacheTag{
				ETag:         etag,
				LastModified: lastModified,
			},
			nil,
		)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/jwks.json", nil)

		handler.GetJWKS(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, etag, w.Header().Get("ETag"))
		assert.Equal(t, lastModified.Format(http.TimeFormat), w.Header().Get("Last-Modified"))
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
		assert.JSONEq(t, string(jwksJSON), w.Body.String())

		mockPublish.AssertExpectations(t)
	})

	t.Run("使用 ETag 缓存返回 304", func(t *testing.T) {
		handler, _, mockPublish := setupTestHandler()

		etag := "test-etag-123"
		mockPublish.On("BuildJWKS", mock.Anything).Return(
			[]byte(`{"keys":[]}`),
			driving.CacheTag{
				ETag:         etag,
				LastModified: time.Now(),
			},
			nil,
		)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
		c.Request.Header.Set("If-None-Match", etag)

		handler.GetJWKS(c)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.String())

		mockPublish.AssertExpectations(t)
	})

	t.Run("构建 JWKS 失败", func(t *testing.T) {
		handler, _, mockPublish := setupTestHandler()

		mockPublish.On("BuildJWKS", mock.Anything).Return(
			nil,
			driving.CacheTag{},
			errors.New("database error"),
		)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/.well-known/jwks.json", nil)

		handler.GetJWKS(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockPublish.AssertExpectations(t)
	})
}

// TestCreateKey 测试创建密钥
func TestCreateKey(t *testing.T) {
	setupGinTest()

	t.Run("成功创建密钥", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		reqBody := `{"algorithm":"RS256"}`
		key := createTestKey("test-kid-123", "RS256", jwks.KeyActive)

		mockMgmt.On("CreateKey", mock.Anything, "RS256", (*time.Time)(nil), (*time.Time)(nil)).Return(key, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys", strings.NewReader(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test-kid-123", response["kid"])
		assert.Equal(t, "RS256", response["algorithm"])

		mockMgmt.AssertExpectations(t)
	})

	t.Run("无效的算法", func(t *testing.T) {
		handler, _, _ := setupTestHandler()

		reqBody := `{"algorithm":"INVALID"}`

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys", strings.NewReader(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("创建密钥失败", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		reqBody := `{"algorithm":"RS256"}`
		mockMgmt.On("CreateKey", mock.Anything, "RS256", (*time.Time)(nil), (*time.Time)(nil)).Return(nil, errors.New("key generation failed"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys", strings.NewReader(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockMgmt.AssertExpectations(t)
	})
}

// TestGetKey 测试获取单个密钥
func TestGetKey(t *testing.T) {
	setupGinTest()

	t.Run("成功获取密钥", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		key := createTestKey("test-kid-123", "RS256", jwks.KeyActive)
		mockMgmt.On("GetKeyByKid", mock.Anything, "test-kid-123").Return(key, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/admin/jwks/keys/test-kid-123", nil)
		c.Params = gin.Params{gin.Param{Key: "kid", Value: "test-kid-123"}}

		handler.GetKey(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test-kid-123", response["kid"])

		mockMgmt.AssertExpectations(t)
	})

	t.Run("密钥不存在", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		mockMgmt.On("GetKeyByKid", mock.Anything, "non-existent").Return(nil, errors.New("key not found"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/admin/jwks/keys/non-existent", nil)
		c.Params = gin.Params{gin.Param{Key: "kid", Value: "non-existent"}}

		handler.GetKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockMgmt.AssertExpectations(t)
	})
}

// TestRetireKey 测试退役密钥
func TestRetireKey(t *testing.T) {
	setupGinTest()

	t.Run("成功退役密钥", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		mockMgmt.On("RetireKey", mock.Anything, "test-kid-123").Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys/test-kid-123/retire", nil)
		c.Params = gin.Params{gin.Param{Key: "kid", Value: "test-kid-123"}}

		handler.RetireKey(c)

		assert.Equal(t, http.StatusNoContent, w.Code)

		mockMgmt.AssertExpectations(t)
	})

	t.Run("退役失败", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		mockMgmt.On("RetireKey", mock.Anything, "test-kid-123").Return(errors.New("cannot retire active key"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys/test-kid-123/retire", nil)
		c.Params = gin.Params{gin.Param{Key: "kid", Value: "test-kid-123"}}

		handler.RetireKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockMgmt.AssertExpectations(t)
	})
}

// TestCleanupExpiredKeys 测试清理过期密钥
func TestCleanupExpiredKeys(t *testing.T) {
	setupGinTest()

	t.Run("成功清理密钥", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		mockMgmt.On("CleanupExpiredKeys", mock.Anything).Return(3, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys/cleanup", nil)

		handler.CleanupExpiredKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(3), response["deleted_count"])

		mockMgmt.AssertExpectations(t)
	})

	t.Run("清理失败", func(t *testing.T) {
		handler, mockMgmt, _ := setupTestHandler()

		mockMgmt.On("CleanupExpiredKeys", mock.Anything).Return(0, errors.New("database error"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/admin/jwks/keys/cleanup", nil)

		handler.CleanupExpiredKeys(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockMgmt.AssertExpectations(t)
	})
}

// TestGetPublishableKeys 测试获取可发布的密钥
func TestGetPublishableKeys(t *testing.T) {
	setupGinTest()

	t.Run("成功获取可发布密钥", func(t *testing.T) {
		handler, _, mockPublish := setupTestHandler()

		keys := []*jwks.Key{
			createTestKey("key-1", "RS256", jwks.KeyActive),
			createTestKey("key-2", "RS384", jwks.KeyGrace),
		}

		mockPublish.On("GetPublishableKeys", mock.Anything).Return(keys, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/admin/jwks/keys/publishable", nil)

		handler.GetPublishableKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		keysArr := response["keys"].([]interface{})
		assert.Equal(t, 2, len(keysArr))

		mockPublish.AssertExpectations(t)
	})

	t.Run("获取失败", func(t *testing.T) {
		handler, _, mockPublish := setupTestHandler()

		mockPublish.On("GetPublishableKeys", mock.Anything).Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/admin/jwks/keys/publishable", nil)

		handler.GetPublishableKeys(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockPublish.AssertExpectations(t)
	})
}
