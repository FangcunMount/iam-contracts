package handler

import (
	"context"
	"errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz/dto"
	"net/http"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/gin-gonic/gin"
)

type OperationQuerier interface {
	Query(context.Context, string) (operation.QueryView, error)
}

type OperationHandler struct{ query OperationQuerier }

func NewOperationHandler(query OperationQuerier) *OperationHandler {
	return &OperationHandler{query: query}
}

// GetOperation returns the authorized original result and producer phases.
// @Summary 查询授权写操作
// @ID getAuthorizationOperation
// @Description Requires the original authenticated principal, scope and current original permissions. Producer commit does not prove consumer completion.
// @Tags Authorization-Operations
// @Produce json
// @Param operation_id path string true "Operation ID returned by an authorization write"
// @Success 200 {object} dto.Response{data=operation.QueryView}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /v4/authz/operations/{operation_id} [get]
func (h *OperationHandler) GetOperation(c *gin.Context) {
	if !operation.ValidID(c.Param("operation_id")) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_operation_id"})
		return
	}
	if h == nil || h.query == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "operation_status_unavailable"})
		return
	}
	view, err := h.query.Query(c.Request.Context(), c.Param("operation_id"))
	if err != nil {
		switch {
		case errors.Is(err, operation.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": "operation_not_found"})
		case errors.Is(err, operation.ErrAccessDenied):
			c.JSON(http.StatusForbidden, gin.H{"code": "operation_access_denied"})
		case errors.Is(err, operation.ErrInvalid):
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "operation_evidence_unavailable"})
		default:
			// Database, policy and protocol errors never expose driver details.
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "operation_status_unavailable"})
		}
		return
	}
	c.JSON(http.StatusOK, dto.NewResponse(view))
}

func operationError(c *gin.Context, err error) bool {
	if errors.Is(err, operation.ErrConflict) || errors.Is(err, transaction.ErrInputConflict) {
		c.JSON(http.StatusConflict, gin.H{"code": "idempotency_conflict", "message": "the key is already bound to different business input"})
		return true
	}
	var outcome *operation.OutcomeError
	if !errors.As(err, &outcome) {
		return false
	}
	status, code := http.StatusServiceUnavailable, "operation_outcome_unknown"
	if outcome.Local == transaction.LocalAborted {
		status, code = http.StatusConflict, "operation_aborted"
	}
	c.JSON(status, gin.H{"code": code, "operation_id": outcome.OperationID, "local": outcome.Local, "message": "inspect the original operation before authorizing another write"})
	return true
}
