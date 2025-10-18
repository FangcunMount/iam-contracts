package code

import (
	"net/http"

	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Authorization errors (102xxx).
const (
	// ErrPermissionDenied - 403: Permission denied.
	ErrPermissionDenied = 100207

	// ErrRoleNotFound - 404: Role not found.
	ErrRoleNotFound = 102001

	// ErrRoleAlreadyExists - 409: Role already exists.
	ErrRoleAlreadyExists = 102002

	// ErrResourceNotFound - 404: Resource not found.
	ErrResourceNotFound = 102003

	// ErrResourceAlreadyExists - 409: Resource already exists.
	ErrResourceAlreadyExists = 102004

	// ErrAssignmentNotFound - 404: Assignment not found.
	ErrAssignmentNotFound = 102005

	// ErrInvalidAction - 400: Invalid action for resource.
	ErrInvalidAction = 102006

	// ErrPolicyVersionNotFound - 404: Policy version not found.
	ErrPolicyVersionNotFound = 102007
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
