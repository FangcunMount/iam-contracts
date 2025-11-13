package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBaseHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewBaseHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := gin.H{"message": "success"}
	h.Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message"`)
}

func TestBaseHandler_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewBaseHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.WithCode(code.ErrNotFoundAccount, "account not found")
	h.Error(c, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code"`)
}

func TestBaseHandler_ErrorWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewBaseHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.ErrorWithCode(c, code.ErrAccountExists, "account %s already exists", "test@example.com")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code"`)
	// 注意: 错误消息使用的是预注册的错误消息模板，不会包含格式化参数
}

func TestBaseHandler_BindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		body      string
		expectErr bool
	}{
		{
			name:      "valid JSON",
			body:      `{"name":"test","value":123}`,
			expectErr: false,
		},
		{
			name:      "invalid JSON",
			body:      `{"name":"test",}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewBaseHandler()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var obj map[string]interface{}
			err := h.BindJSON(c, &obj)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBaseHandler_GetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setValue interface{}
		setKey   string
		expected string
		exists   bool
	}{
		{
			name:     "string user ID",
			setValue: "user123",
			setKey:   "user_id",
			expected: "user123",
			exists:   true,
		},
		{
			name:     "uint64 user ID",
			setValue: uint64(12345),
			setKey:   "user_id",
			expected: "12345",
			exists:   true,
		},
		{
			name:     "int64 user ID",
			setValue: int64(12345),
			setKey:   "user_id",
			expected: "12345",
			exists:   true,
		},
		{
			name:     "no user ID",
			setValue: nil,
			setKey:   "",
			expected: "",
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewBaseHandler()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setKey != "" {
				c.Set(tt.setKey, tt.setValue)
			}

			userID, exists := h.GetUserID(c)
			assert.Equal(t, tt.expected, userID)
			assert.Equal(t, tt.exists, exists)
		})
	}
}

func TestBaseHandler_GetTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setValue string
		setKey   string
		expected string
	}{
		{
			name:     "tenant from context",
			setValue: "tenant123",
			setKey:   "tenant_id",
			expected: "tenant123",
		},
		{
			name:     "default tenant",
			setValue: "",
			setKey:   "",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewBaseHandler()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// 创建一个空的 HTTP 请求，避免 nil pointer
			c.Request = httptest.NewRequest("GET", "/", nil)

			if tt.setKey != "" {
				c.Set(tt.setKey, tt.setValue)
			}

			tenantID := h.GetTenantID(c)
			assert.Equal(t, tt.expected, tenantID)
		})
	}
}

func TestParseUint(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		field     string
		expected  uint64
		expectErr bool
	}{
		{
			name:      "valid uint",
			raw:       "12345",
			field:     "id",
			expected:  12345,
			expectErr: false,
		},
		{
			name:      "empty string",
			raw:       "",
			field:     "id",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "invalid number",
			raw:       "abc",
			field:     "id",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "negative number",
			raw:       "-123",
			field:     "id",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUint(tt.raw, tt.field)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		field     string
		expected  int64
		expectErr bool
	}{
		{
			name:      "valid positive int",
			raw:       "12345",
			field:     "id",
			expected:  12345,
			expectErr: false,
		},
		{
			name:      "valid negative int",
			raw:       "-12345",
			field:     "offset",
			expected:  -12345,
			expectErr: false,
		},
		{
			name:      "empty string",
			raw:       "",
			field:     "id",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "invalid number",
			raw:       "abc",
			field:     "id",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseInt(tt.raw, tt.field)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
