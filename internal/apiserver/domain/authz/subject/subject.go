package subject

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Type 主体类型
type Type string

const (
	TypeUser    Type = "user"    // 用户
	TypeGroup   Type = "group"   // 组
	TypeService Type = "service" // 服务
)

// Ref 引用被分配角色或接受鉴权的主体，不表达身份已验证或主体确实存在。
type Ref struct {
	Type Type    // 主体类型
	ID   meta.ID // 对应主体类型下的主体 ID
}

// NewRef 创建主体引用
func NewRef(subjectType Type, id meta.ID) (Ref, error) {
	subjectType = Type(strings.TrimSpace(string(subjectType)))
	if subjectType == "" {
		return Ref{}, perrors.WithCode(code.ErrInvalidArgument, "subject type is required")
	}
	switch subjectType {
	case TypeUser, TypeGroup, TypeService:
	default:
		return Ref{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported subject type: %s", subjectType)
	}
	if id.IsZero() {
		return Ref{}, perrors.WithCode(code.ErrInvalidArgument, "subject id is required")
	}
	return Ref{Type: subjectType, ID: id}, nil
}

// NewUserRef 创建用户引用
func NewUserRef(id meta.ID) (Ref, error) {
	return NewRef(TypeUser, id)
}

// ParseRef 解析 <type>:<id> 形式的主体引用。
func ParseRef(value string) (Ref, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Ref{}, perrors.WithCode(code.ErrInvalidArgument, "subject must use <type>:<id>")
	}
	id, err := meta.ParseID(parts[1])
	if err != nil {
		return Ref{}, perrors.WithCode(code.ErrInvalidArgument, "subject id is invalid")
	}
	return NewRef(Type(parts[0]), id)
}

// IsZero 判断主体引用是否为空
func (r Ref) IsZero() bool {
	return r.Type == "" || r.ID.IsZero()
}

// String 返回主体引用字符串表示
func (r Ref) String() string {
	if r.IsZero() {
		return ""
	}
	return string(r.Type) + ":" + r.ID.String()
}
