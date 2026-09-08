package assignment

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Assignment 表达主体在租户内持有某个角色的赋权事实（聚合根）。
type Assignment struct {
	ID AssignmentID

	//---- 赋权主体 ----
	SubjectType SubjectType // user/group/service
	SubjectID   meta.ID     // 用户或组ID

	//---- 赋权事实 ----
	RoleID meta.ID // 角色ID
	// 租户ID（域）

	//---- 赋权来源 ----
	GrantedBy string // 授权人
}

// NewAssignment 创建新赋权。
func NewAssignment(subjectType SubjectType, subjectID meta.ID, roleID meta.ID, opts ...Option) (Assignment, error) {
	subjectType = SubjectType(strings.TrimSpace(string(subjectType)))
	a := Assignment{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		RoleID:      roleID,
	}
	for _, opt := range opts {
		opt(&a)
	}
	a.GrantedBy = strings.TrimSpace(a.GrantedBy)
	if _, err := subject.NewRef(subject.Type(subjectType), subjectID); err != nil {
		return Assignment{}, err
	}
	if roleID.IsZero() {
		return Assignment{}, perrors.WithCode(code.ErrInvalidArgument, "role id is required")
	}
	if a.GrantedBy == "" {
		return Assignment{}, perrors.WithCode(code.ErrInvalidArgument, "granted by is required")
	}
	return a, nil
}

// Option 配置赋权事实。
type Option func(*Assignment)

func WithID(id AssignmentID) Option  { return func(a *Assignment) { a.ID = id } }
func WithGrantedBy(by string) Option { return func(a *Assignment) { a.GrantedBy = by } }

// AssignmentID 是赋权事实的标识。
type AssignmentID meta.ID

func NewAssignmentID(value uint64) AssignmentID {
	id := meta.FromUint64(value) // 来自 URL 或内部生成
	return AssignmentID(id)
}

func (id AssignmentID) Uint64() uint64 {
	return meta.ID(id).Uint64()
}

func (id AssignmentID) String() string {
	return meta.ID(id).String()
}

// SubjectType 主体类型
type SubjectType = subject.Type

const (
	SubjectTypeUser    = subject.TypeUser
	SubjectTypeGroup   = subject.TypeGroup
	SubjectTypeService = subject.TypeService
)

func (a Assignment) SubjectTypeString() string {
	return string(a.SubjectType)
}

// Fact states that a subject holds a role inside a tenant.
type Fact struct {
	Subject  subject.Ref
	RoleName role.Name

	GrantedBy string
}

func NewFact(sub subject.Ref, roleName, grantedBy string) (Fact, error) {
	grantedBy = strings.TrimSpace(grantedBy)
	if sub.IsZero() {
		return Fact{}, perrors.WithCode(code.ErrInvalidArgument, "subject is required")
	}
	roleNameValue, err := role.NewName(roleName)
	if err != nil {
		return Fact{}, err
	}
	return Fact{Subject: sub, RoleName: roleNameValue, GrantedBy: grantedBy}, nil
}

func (f Fact) RoleNameString() string {
	return f.RoleName.String()
}
