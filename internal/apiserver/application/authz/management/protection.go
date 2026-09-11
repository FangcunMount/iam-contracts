// Package management 落实授权管理的角色保护边界。
package management

import (
	"context"
	"slices"

	admission "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignmentadmission"

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

type Guard struct {
	checker     Checker
	assignments admission.Policy
}

func NewGuard(checker Checker, policies ...admission.Policy) Guard {
	g := Guard{checker: checker}
	if len(policies) > 0 {
		g.assignments = policies[0]
	}
	return g
}

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
	request, err := authorization.NewRequest(a.user, role.ManageProtectedResource, role.ManageProtectedAction)
	if err != nil {
		return false, err
	}
	decision, err := g.checker.Check(ctx, request)
	return decision.Allowed, err
}

func (g Guard) Require(ctx context.Context, roles ...*role.Role) error {
	needsProtection := false
	for _, r := range roles {
		if r == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "角色不存在")
		}
		if err := r.ManagementProtection.Validate(); err != nil {
			return err
		}
		needsProtection = needsProtection || r.IsProtected()
	}
	if needsProtection {
		allowed, err := g.CanManageProtected(ctx)
		if err != nil {
			return err
		}
		if !allowed {
			return perrors.WithCode(code.ErrPermissionDenied, "操作受保护角色需要专门管理权限")
		}
	}
	return nil
}

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

// RequireOperation 在应用入口校验原始操作权限；服务身份只接受可信上下文。
func (g Guard) RequireOperation(ctx context.Context, resource, action string) error {
	a, ok := ctx.Value(actorKey{}).(actor)
	if ok && a.service == "admin" {
		return nil
	}
	if !ok || a.service != "" || a.user.IsZero() || g.checker == nil {
		return deniedOperation()
	}
	request, err := authorization.NewRequest(a.user, resource, action)
	if err != nil {
		return err
	}
	decision, err := g.checker.Check(ctx, request)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return deniedOperation()
	}
	return nil
}
func deniedOperation() error {
	return perrors.WithCode(code.ErrPermissionDenied, "授权管理操作不被允许")
}

func (g Guard) RequireAssignment(ctx context.Context, sub subject.Ref, target *role.Role, operation admission.Operation, changedBy string) error {
	a, ok := ctx.Value(actorKey{}).(actor)
	if !ok {
		return deniedOperation()
	}
	if a.service != "" && a.service != "admin" {
		if g.assignments == nil || target == nil {
			return deniedOperation()
		}
		err := g.assignments.AuthorizeAssignment(admission.Request{CallerService: a.service, Subject: sub, RoleName: target.Name, Operation: operation, DelegatedActor: changedBy})
		if err != nil {
			return deniedOperation()
		}
	} else if err := g.RequireOperation(ctx, "iam:authz:collection:assignments", string(operation)); err != nil {
		return err
	}
	return g.Require(ctx, target)
}

// RequireReplacement 从部署策略重建管理集合，禁止命令自行扩大 Replace 范围。
func (g Guard) RequireReplacement(ctx context.Context, sub subject.Ref, names, managed []string, changedBy string) error {
	a, ok := ctx.Value(actorKey{}).(actor)
	if !ok {
		return deniedOperation()
	}
	if a.service == "" || a.service == "admin" {
		if err := g.RequireOperation(ctx, "iam:authz:collection:assignments", "grant"); err != nil {
			return err
		}
		return g.RequireOperation(ctx, "iam:authz:collection:assignments", "revoke")
	}
	if g.assignments == nil {
		return deniedOperation()
	}
	roleNames := make([]role.Name, 0, len(names))
	for _, name := range names {
		n, err := role.NewName(name)
		if err != nil {
			return err
		}
		roleNames = append(roleNames, n)
	}
	allowed, err := g.assignments.AuthorizeReplacement(admission.ReplacementRequest{CallerService: a.service, Subject: sub, RoleNames: roleNames, DelegatedActor: changedBy})
	if err != nil {
		return deniedOperation()
	}
	actual := slices.Clone(managed)
	slices.Sort(actual)
	actual = slices.Compact(actual)
	slices.Sort(allowed)
	if !slices.Equal(actual, allowed) {
		return deniedOperation()
	}
	return nil
}
