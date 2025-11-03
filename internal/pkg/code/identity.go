package code

import (
	"net/http"

	"github.com/FangcunMount/component-base/pkg/errors"
)

// Identity: 基础用户及身份档案/监护等领域错误码 (101000～101999).

// Identity: 用户基础错误 (101000～101099).
const (
	// ErrUserNotFound - 404: User not found.
	ErrUserNotFound = 101000

	// ErrUserAlreadyExists - 400: User already exist.
	ErrUserAlreadyExists = 101001

	// ErrUserBasicInfoInvalid - 400: User basic info is invalid.
	ErrUserBasicInfoInvalid = 101002

	// ErrUserStatusInvalid - 400: User status is invalid.
	ErrUserStatusInvalid = 101003

	// ErrUserInvalid - 400: User is invalid.
	ErrUserInvalid = 101004

	// ErrUserBlocked - 403: User is blocked.
	ErrUserBlocked = 101005

	// ErrUserInactive - 403: User is inactive.
	ErrUserInactive = 101006
)

// Identity: 儿童档案错误 (101100～101199).
const (
	// ErrIdentityChildExists - 400: 儿童档案已存在
	ErrIdentityChildExists = 101100

	// ErrIdentityChildNotFound - 404: 儿童不存在
	ErrIdentityChildNotFound = 101101
)

// Identity: 监护关系错误 (101200～101299).
const (
	// ErrIdentityGuardianshipExists - 400: 监护关系已存在
	ErrIdentityGuardianshipExists = 101200

	// ErrIdentityGuardianshipNotFound - 404: 监护关系不存在
	ErrIdentityGuardianshipNotFound = 101201
)

// nolint: gochecknoinits
func init() {
	registerIdentity()
}

func registerIdentity() {
	// 用户基础错误
	errors.MustRegister(&identityCoder{code: ErrUserNotFound, status: http.StatusNotFound, msg: "User not found"})
	errors.MustRegister(&identityCoder{code: ErrUserAlreadyExists, status: http.StatusBadRequest, msg: "User already exist"})
	errors.MustRegister(&identityCoder{code: ErrUserBasicInfoInvalid, status: http.StatusBadRequest, msg: "User basic info is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserStatusInvalid, status: http.StatusBadRequest, msg: "User status is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserInvalid, status: http.StatusBadRequest, msg: "User is invalid"})
	errors.MustRegister(&identityCoder{code: ErrUserBlocked, status: http.StatusForbidden, msg: "User is blocked"})
	errors.MustRegister(&identityCoder{code: ErrUserInactive, status: http.StatusForbidden, msg: "User is inactive"})

	// 儿童档案错误
	errors.MustRegister(&identityCoder{code: ErrIdentityChildExists, status: http.StatusBadRequest, msg: "儿童档案已存在"})
	errors.MustRegister(&identityCoder{code: ErrIdentityChildNotFound, status: http.StatusNotFound, msg: "儿童不存在"})

	// 监护关系错误
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
