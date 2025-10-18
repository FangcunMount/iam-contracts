package code

import (
	"net/http"

	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Base user & identity module errors.
const (
	// ErrUserNotFound - 404: User not found.
	ErrUserNotFound = 110001

	// ErrUserAlreadyExists - 400: User already exist.
	ErrUserAlreadyExists = 110002

	// ErrUserBasicInfoInvalid - 400: User basic info is invalid.
	ErrUserBasicInfoInvalid = 110003

	// ErrUserStatusInvalid - 400: User status is invalid.
	ErrUserStatusInvalid = 110004

	// ErrUserInvalid - 400: User is invalid.
	ErrUserInvalid = 110005

	// ErrUserBlocked - 403: User is blocked.
	ErrUserBlocked = 110006

	// ErrUserInactive - 403: User is inactive.
	ErrUserInactive = 110007
)

// Identity (child, guardianship) module errors (110101+).
const (
	// ErrIdentityUserBlocked - 403: 用户被封禁
	ErrIdentityUserBlocked = 110101

	// ErrIdentityChildExists - 400: 儿童档案已存在
	ErrIdentityChildExists = 110102

	// ErrIdentityChildNotFound - 404: 儿童不存在
	ErrIdentityChildNotFound = 110103

	// ErrIdentityGuardianshipExists - 400: 监护关系已存在
	ErrIdentityGuardianshipExists = 110104

	// ErrIdentityGuardianshipNotFound - 404: 监护关系不存在
	ErrIdentityGuardianshipNotFound = 110105
)

// nolint: gochecknoinits
func init() {
	registerIdentity()
}

func registerIdentity() {
	// Base user & identity module errors
	errors.MustRegister(&identityCoder{code: ErrUserNotFound, status: http.StatusNotFound, msg: "User not found"})
	errors.MustRegister(&identityCoder{code: ErrUserAlreadyExists, status: http.StatusBadRequest, msg: "User already exist"})
	errors.MustRegister(&identityCoder{code: ErrUserBasicInfoInvalid, status: http.StatusBadRequest, msg: "User basic info is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserStatusInvalid, status: http.StatusBadRequest, msg: "User status is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserInvalid, status: http.StatusBadRequest, msg: "User is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserBlocked, status: http.StatusForbidden, msg: "User is blocked"})
	errors.MustRegister(&identityCoder{code: ErrUserInactive, status: http.StatusForbidden, msg: "User is inactive"})

	// Identity (child, guardianship) module errors
	errors.MustRegister(&identityCoder{code: ErrIdentityUserBlocked, status: http.StatusForbidden, msg: "用户被封禁"})
	errors.MustRegister(&identityCoder{code: ErrIdentityChildExists, status: http.StatusBadRequest, msg: "儿童档案已存在"})
	errors.MustRegister(&identityCoder{code: ErrIdentityChildNotFound, status: http.StatusNotFound, msg: "儿童不存在"})
	errors.MustRegister(&identityCoder{code: ErrIdentityGuardianshipExists, status: http.StatusBadRequest, msg: "监护关系已存在"})
	errors.MustRegister(&identityCoder{code: ErrIdentityGuardianshipNotFound, status: http.StatusNotFound, msg: "监护关系不存在"})
}

// identityCoder 实现 errors.Coder 接口
type identityCoder struct {
	code   int
	status int
	msg    string
}

func (c *identityCoder) Code() int {
	return c.code
}

func (c *identityCoder) HTTPStatus() int {
	return c.status
}

func (c *identityCoder) String() string {
	return c.msg
}

func (c *identityCoder) Reference() string {
	return ""
}
