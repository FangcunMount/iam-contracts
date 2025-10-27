package code

import (
	"net/http"

	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// Authorization errors (103xxx - changed from 102xxx to avoid conflict with JWKS).
const (
	// ErrPermissionDenied - 403: Permission denied.
	ErrPermissionDenied = 100207

	// ErrRoleNotFound - 404: Role not found.
	ErrRoleNotFound = 103001

	// ErrRoleAlreadyExists - 409: Role already exists.
	ErrRoleAlreadyExists = 103002

	// ErrResourceNotFound - 404: Resource not found.
	ErrResourceNotFound = 103003

	// ErrResourceAlreadyExists - 409: Resource already exists.
	ErrResourceAlreadyExists = 103004

	// ErrAssignmentNotFound - 404: Assignment not found.
	ErrAssignmentNotFound = 103005

	// ErrInvalidAction - 400: Invalid action for resource.
	ErrInvalidAction = 103006

	// ErrPolicyVersionNotFound - 404: Policy version not found.
	ErrPolicyVersionNotFound = 103007
)

// nolint: gochecknoinits
func init() {
	registerAuthz()
}

func registerAuthz() {
	registerAuthzCode(ErrPermissionDenied, http.StatusForbidden, "Permission denied")
	registerAuthzCode(ErrRoleNotFound, http.StatusNotFound, "Role not found")
	registerAuthzCode(ErrRoleAlreadyExists, http.StatusConflict, "Role already exists")
	registerAuthzCode(ErrResourceNotFound, http.StatusNotFound, "Resource not found")
	registerAuthzCode(ErrResourceAlreadyExists, http.StatusConflict, "Resource already exists")
	registerAuthzCode(ErrAssignmentNotFound, http.StatusNotFound, "Assignment not found")
	registerAuthzCode(ErrInvalidAction, http.StatusBadRequest, "Invalid action for resource")
	registerAuthzCode(ErrPolicyVersionNotFound, http.StatusNotFound, "Policy version not found")
}

func registerAuthzCode(code int, httpStatus int, message string) {
	errors.MustRegister(&authzCoder{
		code:   code,
		status: httpStatus,
		msg:    message,
	})
}

// authzCoder 实现 errors.Coder 接口
type authzCoder struct {
	code   int
	status int
	msg    string
}

func (c *authzCoder) Code() int {
	return c.code
}

func (c *authzCoder) HTTPStatus() int {
	return c.status
}

func (c *authzCoder) String() string {
	return c.msg
}

func (c *authzCoder) Reference() string {
	return ""
}
