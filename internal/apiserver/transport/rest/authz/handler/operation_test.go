package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type operationQueryFunc func(context.Context, string) (operation.QueryView, error)

func (f operationQueryFunc) Query(ctx context.Context, id string) (operation.QueryView, error) {
	return f(ctx, id)
}

func TestOperationHTTPDoesNotExposePrivateErrorsOrClaimRollbackOnUnknown(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{operation.ErrNotFound, 404, "operation_not_found"}, {operation.ErrAccessDenied, 403, "operation_access_denied"},
		{operation.ErrInvalid, 503, "operation_evidence_unavailable"}, {errors.New("private DSN wire token"), 503, "operation_status_unavailable"},
	} {
		engine := gin.New()
		h := NewOperationHandler(operationQueryFunc(func(context.Context, string) (operation.QueryView, error) { return operation.QueryView{}, tc.err }))
		engine.GET("/operations/:operation_id", h.GetOperation)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest("GET", "/operations/f46ebdfd-c2a4-43b9-aafb-b98b9675505d", nil))
		require.Equal(t, tc.status, response.Code)
		require.Contains(t, response.Body.String(), tc.code)
		require.NotContains(t, response.Body.String(), "private")
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{operation.ErrConflict, 409, "idempotency_conflict"},
		{&operation.OutcomeError{OperationID: "original", Local: transaction.LocalUnknown, Cause: errors.New("private token")}, 503, "operation_outcome_unknown"},
		{&operation.OutcomeError{OperationID: "original", Local: transaction.LocalAborted}, 409, "operation_aborted"},
	} {
		engine := gin.New()
		engine.POST("/write", func(c *gin.Context) { handleError(c, tc.err) })
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest("POST", "/write", nil))
		require.Equal(t, tc.status, response.Code)
		require.Contains(t, response.Body.String(), tc.code)
		require.NotContains(t, response.Body.String(), "private")
	}
}
