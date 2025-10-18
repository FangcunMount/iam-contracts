package code

import (
	"net/http"

	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Authorization errors (100207).
const (
	// ErrPermissionDenied - 403: Permission denied.
	ErrPermissionDenied = 100207
)

// nolint: gochecknoinits
func init() {
	registerAuthz()
}

func registerAuthz() {
	errors.MustRegister(&authzCoder{code: ErrPermissionDenied, status: http.StatusForbidden, msg: "Permission denied"})
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
