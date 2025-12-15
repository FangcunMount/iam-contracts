package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	roleDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	roleMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/role"
)

// ==================== 角色 Seed 函数 ====================

// seedRoles 创建角色
//
// 业务说明：
// 1. 创建配置中的所有角色
// 2. 返回的 state 保存角色ID，供后续步骤使用（如 assignments 步骤）
//
// 幂等性：通过角色名称查询检查，已存在的角色会被更新而不是重复创建
func seedRoles(ctx context.Context, deps *dependencies, state *seedContext) error {
	if deps.Config == nil || len(deps.Config.Roles) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有角色数据，跳过")
		return nil
	}

	// 初始化角色仓储
	roleRepo := roleMysql.NewRoleRepository(deps.DB)

	for _, rc := range deps.Config.Roles {
		id, err := ensureRole(ctx, roleRepo, rc)
		if err != nil {
			return fmt.Errorf("ensure role %s: %w", rc.Alias, err)
		}
		state.Roles[rc.Alias] = id
		deps.Logger.Infow("✅ 角色创建/更新成功",
			"alias", rc.Alias,
			"name", rc.Name,
			"role_id", id)
	}

	deps.Logger.Infow("✅ 角色初始化完成", "count", len(deps.Config.Roles))
	return nil
}

// ensureRole 确保角色存在（如不存在则创建，如存在则更新）
func ensureRole(
	ctx context.Context,
	roleRepo roleDomain.Repository,
	cfg RoleConfig,
) (string, error) {
	// 先尝试通过名称查询
	existing, err := roleRepo.FindByName(ctx, cfg.TenantID, cfg.Name)
	if err == nil && existing != nil {
		// 角色已存在，更新信息
		if existing.DisplayName != cfg.DisplayName || existing.Description != cfg.Description {
			existing.DisplayName = cfg.DisplayName
			existing.Description = cfg.Description
			if err := roleRepo.Update(ctx, existing); err != nil {
				return "", fmt.Errorf("update role: %w", err)
			}
		}
		return strconv.FormatUint(uint64(existing.ID), 10), nil
	}

	// 角色不存在，创建新角色
	newRole := roleDomain.NewRole(
		cfg.Name,
		cfg.DisplayName,
		cfg.TenantID,
		roleDomain.WithDescription(cfg.Description),
	)

	if err := roleRepo.Create(ctx, &newRole); err != nil {
		return "", fmt.Errorf("create role: %w", err)
	}

	return strconv.FormatUint(uint64(newRole.ID), 10), nil
}

// ==================== 辅助函数 ====================

// resolveUserAlias 解析用户别名（支持 @alias 引用）
// 如果以 @ 开头，从 state.Users 中查找；否则直接返回原值
func resolveUserAlias(subjectID string, state *seedContext) string {
	if strings.HasPrefix(subjectID, "@") {
		alias := strings.TrimPrefix(subjectID, "@")
		if userID, ok := state.Users[alias]; ok {
			return userID
		}
	}
	return subjectID
}

// resolveRoleAlias 解析角色别名，返回角色ID
// 如果提供了 roleAlias（@开头），从 state.Roles 中查找
// 否则使用 roleID
func resolveRoleAlias(roleID uint64, roleAlias string, state *seedContext) (uint64, error) {
	if roleAlias != "" {
		alias := strings.TrimPrefix(roleAlias, "@")
		if id, ok := state.Roles[alias]; ok {
			return strconv.ParseUint(id, 10, 64)
		}
		return 0, fmt.Errorf("role alias not found: %s", alias)
	}
	if roleID > 0 {
		return roleID, nil
	}
	return 0, fmt.Errorf("neither role_id nor role_alias provided")
}
