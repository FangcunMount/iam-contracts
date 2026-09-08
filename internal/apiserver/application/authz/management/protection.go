// Package management 落实授权管理的角色保护边界。
package management

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

type actorKey struct{}
type actor struct {
	user    subject.Ref
	service string
}

// WithAuthenticatedUser 仅由已完成身份验证的传输适配器或受信任内部调用方设置。
func WithAuthenticatedUser(ctx context.Context, user subject.Ref) context.Context {
	return context.WithValue(ctx, actorKey{}, actor{user: user})
}

// WithAuthenticatedService 必须使用双向 TLS 验证得到的服务身份，不能使用请求参数。
func WithAuthenticatedService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor{service: service})
}

type Checker interface {
	Check(context.Context, authorization.Request) (authorization.Decision, error)
}

type Guard struct{ checker Checker }

func NewGuard(checker Checker) Guard { return Guard{checker: checker} }

func (g Guard) CanManageProtected(ctx context.Context) (bool, error) {
	a, ok := ctx.Value(actorKey{}).(actor)
	if !ok {
		return false, nil
	}
	if a.service != "" {
		return a.service == "admin", nil
	}
	if a.user.ID.IsZero() || g.checker == nil {
		return false, nil
	}
	request, err := authorization.NewRequest(a.user, role.ManageProtectedResource, role.ManageProtectedAction, authorization.ObjectContext{})
	if err != nil {
		return false, err
	}
	decision, err := g.checker.Check(ctx, request)
	return decision.Allowed, err
}

func (g Guard) Require(ctx context.Context, roles ...*role.Role) error {
	for _, r := range roles {
		if r == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "角色不存在")
		}
		if err := r.ManagementProtection.Validate(); err != nil {
			return err
		}
		if r.IsProtected() {
			allowed, err := g.CanManageProtected(ctx)
			if err != nil {
				return err
			}
			if !allowed {
				return perrors.WithCode(code.ErrPermissionDenied, "操作受保护角色需要专门管理权限")
			}
			return nil
		}
	}
	return nil
}

func GuardFrom(source any) Guard { checker, _ := source.(Checker); return NewGuard(checker) }

func (g Guard) Visible(ctx context.Context, target *role.Role) (bool, error) {
	if target == nil {
		return false, perrors.WithCode(code.ErrRoleNotFound, "角色不存在")
	}
	if err := target.ManagementProtection.Validate(); err != nil {
		return false, err
	}
	if !target.IsProtected() {
		return true, nil
	}
	return g.CanManageProtected(ctx)
}
func (g Guard) RequireVisible(ctx context.Context, target *role.Role) error {
	allowed, err := g.Visible(ctx, target)
	if err != nil {
		return err
	}
	if !allowed {
		return perrors.WithCode(code.ErrRoleNotFound, "角色不存在")
	}
	return nil
}
