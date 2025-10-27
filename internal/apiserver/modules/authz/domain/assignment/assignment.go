package assignment

import "github.com/FangcunMount/component-base/pkg/util/idutil"

// Assignment 用户/组 ↔ 角色赋权（聚合根）
type Assignment struct {
	ID          AssignmentID
	SubjectType SubjectType // user/group
	SubjectID   string      // 用户或组ID
	RoleID      uint64      // 角色ID
	TenantID    string      // 租户ID（域）
	GrantedBy   string      // 授权人
}

// NewAssignment 创建新赋权
func NewAssignment(subjectType SubjectType, subjectID string, roleID uint64, tenantID string, opts ...AssignmentOption) Assignment {
	a := Assignment{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		RoleID:      roleID,
		TenantID:    tenantID,
	}
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

// AssignmentOption 赋权选项
type AssignmentOption func(*Assignment)

func WithID(id AssignmentID) AssignmentOption  { return func(a *Assignment) { a.ID = id } }
func WithGrantedBy(by string) AssignmentOption { return func(a *Assignment) { a.GrantedBy = by } }

// SubjectKey 返回 Casbin 中的主体标识
func (a *Assignment) SubjectKey() string {
	return string(a.SubjectType) + ":" + a.SubjectID
}

// RoleKey 返回 Casbin 中的角色标识
func (a *Assignment) RoleKey() string {
	return "role:" + idutil.NewID(a.RoleID).String()
}

// AssignmentID 赋权ID值对象
type AssignmentID idutil.ID

func NewAssignmentID(value uint64) AssignmentID {
	return AssignmentID(idutil.NewID(value))
}

func (id AssignmentID) Uint64() uint64 {
	return idutil.ID(id).Uint64()
}

func (id AssignmentID) String() string {
	return idutil.ID(id).String()
}

// SubjectType 主体类型
type SubjectType string

const (
	SubjectTypeUser  SubjectType = "user"
	SubjectTypeGroup SubjectType = "group"
)

func (st SubjectType) String() string {
	return string(st)
}
