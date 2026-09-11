package grant

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Grant 权限授予实体
type Grant struct {
	ID meta.ID // 授予事实ID

	// ---- 授予内容 ----
	RoleID      meta.ID             // 角色ID
	ResourceID  resource.ResourceID // 资源ID
	ResourceKey resource.Key        // 授予的资源键或通配范围；目录关联规则由 ValidateAgainst 校验
	Action      resource.Action     // 授予动作；普通赋权仅允许具体动作，系统赋权可用 *
	Constraints constraint.Set      // 约束集合

	// ---- 内容标识 ----
	GrantKey string // 根据授予内容计算的规范化唯一键

	// ---- 授予来源 ----
	GrantedBy string    // 授予者标识，可来自用户或内部引导过程
	GrantedAt time.Time // 授权时间

	// ---- 撤销状态与记录版本 ----
	RevokedAt *time.Time // 撤销时间
	Version   uint32     // 授予记录版本，非全局授权事实版本
}

func New(
	roleID meta.ID,

	resourceID resource.ResourceID,
	resourceKey string,
	action string,
	constraints constraint.Set,
	grantedBy string,
) (Grant, error) {
	if resourceID.Uint64() == 0 {
		return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "resource id is required for managed permission grant")
	}
	concreteAction, err := resource.NewAction(action)
	if err != nil {
		return Grant{}, err
	}
	if err := concreteAction.ValidateConcrete(); err != nil {
		return Grant{}, err
	}
	return newGrant(roleID, resourceID, resourceKey, concreteAction.String(), constraints, grantedBy, false)
}

// NewSystem 创建可信引导或迁移使用的无条件权限授予。
// 新建授权时，只有此入口允许资源 ID 为空或动作使用通配符；不接受条件约束。
func NewSystem(
	roleID meta.ID,

	resourceID resource.ResourceID,
	resourceKey string,
	action string,
	constraints constraint.Set,
	grantedBy string,
) (Grant, error) {
	return newGrant(roleID, resourceID, resourceKey, action, constraints, grantedBy, true)
}

// newGrant 创建权限授予实体
func newGrant(
	roleID meta.ID,
	resourceID resource.ResourceID,
	resourceKey string,
	action string,
	constraints constraint.Set,
	grantedBy string,
	system bool,
) (Grant, error) {
	if roleID.IsZero() {
		return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "role id is required")
	}
	key, err := resource.NewKey(resourceKey)
	if err != nil {
		return Grant{}, err
	}
	constraints, err = constraints.Normalize()
	if err != nil {
		return Grant{}, err
	}
	action = strings.TrimSpace(action)
	if system {
		if !constraints.IsUnconditional() {
			return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "system wildcard grants must be unconditional")
		}
		if action != resource.WildcardAction.String() {
			concrete, concreteErr := resource.NewAction(action)
			if concreteErr != nil {
				return Grant{}, concreteErr
			}
			if err := concrete.ValidateConcrete(); err != nil {
				return Grant{}, err
			}
			action = concrete.String()
		}
	} else {
		if resourceID.Uint64() == 0 {
			return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "conditional and managed grants require a catalog resource")
		}
		concrete, concreteErr := resource.NewAction(action)
		if concreteErr != nil {
			return Grant{}, concreteErr
		}
		if err := concrete.ValidateConcrete(); err != nil {
			return Grant{}, err
		}
		action = concrete.String()
	}
	if !constraints.IsUnconditional() && isBulkAction(action) {
		return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "conditional grants cannot authorize list, search, or batch actions")
	}
	actionValue, err := resource.NewAction(action)
	if err != nil {
		return Grant{}, err
	}
	grantedBy = strings.TrimSpace(grantedBy)
	if grantedBy == "" {
		return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "granted by is required")
	}
	grant := Grant{
		RoleID:      roleID,
		ResourceID:  resourceID,
		ResourceKey: key,
		Action:      actionValue,
		Constraints: constraints,
		GrantedBy:   grantedBy,
		Version:     1,
	}
	grant.GrantKey, err = grant.computeKey()
	if err != nil {
		return Grant{}, err
	}
	return grant, nil
}

func isBulkAction(action string) bool {
	return action == "list" || action == "search" || action == "batch" || strings.HasPrefix(action, "batch_")
}

func (g Grant) IsActive() bool { return g.RevokedAt == nil }

func (g Grant) IsConditional() bool { return !g.Constraints.IsUnconditional() }

func (g Grant) MatchesAction(action resource.Action) bool {
	return g.Action.String() == resource.WildcardAction.String() || g.Action.Matches(action)
}

func (g Grant) CoversResource(candidate resource.Key) bool {
	return g.ResourceKey.Covers(candidate)
}

func (g *Grant) Revoke(at time.Time) error {
	if g == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "permission grant is required")
	}
	if g.RevokedAt != nil {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	g.RevokedAt = &at
	return nil
}

func (g Grant) ResourceKeyString() string { return g.ResourceKey.String() }

func (g Grant) ActionString() string { return g.Action.String() }

func (g Grant) CanonicalConstraintJSON() ([]byte, error) {
	return g.Constraints.CanonicalJSON()
}

func (g Grant) computeKey() (string, error) {
	constraints, err := g.CanonicalConstraintJSON()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("v2\x00%d\x00%d\x00%s\x00%s\x00%s",
		g.RoleID.Uint64(),
		g.ResourceID.Uint64(),
		g.ResourceKeyString(),
		g.ActionString(),
		constraints,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}

type RestoreOptions struct {
	ID        meta.ID
	GrantKey  string
	GrantedAt time.Time
	RevokedAt *time.Time
	Version   uint32
}

func Restore(
	roleID meta.ID,

	resourceID resource.ResourceID,
	resourceKey string,
	action string,
	constraints constraint.Set,
	grantedBy string,
	options RestoreOptions,
) (Grant, error) {
	system := resourceID.Uint64() == 0 || action == resource.WildcardAction.String()
	grant, err := newGrant(roleID, resourceID, resourceKey, action, constraints, grantedBy, system)
	if err != nil {
		return Grant{}, err
	}
	if options.GrantKey != "" && options.GrantKey != grant.GrantKey {
		return Grant{}, perrors.WithCode(code.ErrInvalidArgument, "permission grant canonical key mismatch")
	}
	grant.ID = options.ID
	grant.GrantedAt = options.GrantedAt
	grant.RevokedAt = options.RevokedAt
	if options.Version > 0 {
		grant.Version = options.Version
	}
	return grant, nil
}

func (g Grant) Clone() Grant {
	out := g
	out.Constraints = g.Constraints.Clone()
	if g.RevokedAt != nil {
		value := *g.RevokedAt
		out.RevokedAt = &value
	}
	return out
}
