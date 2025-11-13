package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestWriteResponse_WithError 测试错误响应统一返回 HTTP 200
func TestWriteResponse_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "NotFound error should return 200",
			err:            errors.WithCode(code.ErrNotFoundAccount, "account not found"),
			expectedStatus: http.StatusOK,
			expectedCode:   code.ErrNotFoundAccount,
		},
		{
			name:           "Unauthorized error should return 200",
			err:            errors.WithCode(code.ErrUnauthenticated, "authentication failed"),
			expectedStatus: http.StatusOK,
			expectedCode:   code.ErrUnauthenticated,
		},
		{
			name:           "Conflict error should return 200",
			err:            errors.WithCode(code.ErrAccountExists, "account already exists"),
			expectedStatus: http.StatusOK,
			expectedCode:   code.ErrAccountExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			WriteResponse(c, tt.err, nil)

			assert.Equal(t, tt.expectedStatus, w.Code, "HTTP status code should be 200")
			assert.Contains(t, w.Body.String(), `"code"`, "Response should contain code field")
		})
	}
}

// TestWriteResponse_WithSuccess 测试成功响应返回 HTTP 200
func TestWriteResponse_WithSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"message": "success"}
	WriteResponse(c, nil, data)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")
	assert.Contains(t, w.Body.String(), `"message"`, "Response should contain data")
}
