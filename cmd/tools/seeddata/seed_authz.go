package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	assignmentApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/application/assignment"
	resourceApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/application/resource"
	assignmentDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	assignmentDriving "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driving"
	assignmentService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/service"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	resourceDriving "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driving"
	resourceService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource/service"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/infra/casbin"
	assignmentMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/assignment"
	resourceMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/resource"
	roleMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/role"
)

// ==================== 授权相关类型定义 ====================

// ==================== 授权资源 Seed 函数 ====================

// seedAuthzResources 创建授权资源数据
//
// 业务说明：
// - 创建系统基础资源定义
// - 每个资源包含允许的操作列表
// - 资源用于后续的权限策略配置
//
// 幂等性：通过资源键查询，已存在的资源会跳过创建
func seedAuthzResources(ctx context.Context, deps *dependencies, state *seedContext) error {
	config := deps.Config
	if config == nil || len(config.Resources) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有资源数据，跳过")
		return nil
	}

	resourceRepo := resourceMysql.NewResourceRepository(deps.DB)
	resourceManager := resourceService.NewResourceManager(resourceRepo)
	resourceCommander := resourceApp.NewResourceCommandService(resourceManager, resourceRepo)
	resourceQueryer := resourceApp.NewResourceQueryService(resourceRepo)

	for _, rc := range config.Resources {
		// 检查资源是否已存在
		if res, err := resourceQueryer.GetResourceByKey(ctx, rc.Key); err == nil && res != nil {
			state.Resources[rc.Alias] = res.ID.Uint64()
			continue
		}

		// 创建新资源
		created, err := resourceCommander.CreateResource(ctx, resourceDriving.CreateResourceCommand{
			Key:         rc.Key,
			DisplayName: rc.DisplayName,
			AppName:     rc.AppName,
			Domain:      rc.Domain,
			Type:        rc.Type,
			Actions:     rc.Actions,
			Description: rc.Description,
		})
		if err != nil {
			return fmt.Errorf("create resource %s: %w", rc.Key, err)
		}
		state.Resources[rc.Alias] = created.ID.Uint64()
	}

	deps.Logger.Infow("✅ 授权资源数据已创建", "count", len(config.Resources))
	return nil
}

// ==================== 角色分配 Seed 函数 ====================

// seedRoleAssignments 创建角色分配数据
//
// 业务说明：
// - 为用户分配系统角色
// - 角色决定用户在系统中的权限
// - 同时在 Casbin 中添加角色继承关系
//
// 前置条件：必须先创建用户和资源
// 幂等性：重复的角色分配会被忽略
func seedRoleAssignments(ctx context.Context, deps *dependencies, state *seedContext) error {
	config := deps.Config
	if config == nil || len(config.Assignments) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有角色分配数据，跳过")
		return nil
	}

	modelPath := deps.CasbinModel
	if _, err := os.Stat(modelPath); err != nil {
		return fmt.Errorf("casbin model file not found: %w", err)
	}

	casbinPort, err := casbin.NewCasbinAdapter(deps.DB, modelPath)
	if err != nil {
		return fmt.Errorf("init casbin adapter: %w", err)
	}

	roleRepo := roleMysql.NewRoleRepository(deps.DB)
	assignmentRepo := assignmentMysql.NewAssignmentRepository(deps.DB)
	assignmentManager := assignmentService.NewAssignmentManager(assignmentRepo, roleRepo)
	assignmentCommander := assignmentApp.NewAssignmentCommandService(assignmentManager, assignmentRepo, casbinPort)

	for _, ac := range config.Assignments {
		// 解析 subject_id: 如果以数字开头,直接使用;否则从 state.Users 查找别名
		subjectID := ac.SubjectID
		if _, ok := state.Users[ac.SubjectID]; ok {
			// 是用户别名,从 state 获取实际ID
			subjectID = state.Users[ac.SubjectID]
		}
		// 否则直接使用配置中的 ID (兼容直接配置ID的情况)

		cmd := assignmentDriving.GrantCommand{
			SubjectType: assignmentDomain.SubjectTypeUser,
			SubjectID:   subjectID,
			RoleID:      ac.RoleID,
			TenantID:    ac.TenantID,
			GrantedBy:   ac.GrantedBy,
		}

		if _, err := assignmentCommander.Grant(ctx, cmd); err != nil {
			if !isDuplicateAssignment(err) {
				return fmt.Errorf("grant role %d to subject %s: %w", ac.RoleID, subjectID, err)
			}
		}
	}

	deps.Logger.Infow("✅ 角色分配数据已创建", "count", len(config.Assignments))
	return nil
}

// ==================== Casbin 策略 Seed 函数 ====================

// seedCasbinPolicies 创建 Casbin 策略规则
//
// 业务说明：
// - 初始化基础的 RBAC 策略规则
// - 定义角色的资源访问权限
// - 设置角色继承关系
//
// 幂等性：Casbin 会自动去重，重复添加不会报错
func seedCasbinPolicies(ctx context.Context, deps *dependencies) error {
	casbinPort, err := casbin.NewCasbinAdapter(deps.DB, deps.CasbinModel)
	if err != nil {
		return fmt.Errorf("init casbin adapter: %w", err)
	}

	// 定义策略规则：角色对资源的访问权限
	policies := []policyDomain.PolicyRule{
		{Sub: "role:super_admin", Dom: "default", Obj: "*", Act: "*"},
		{Sub: "role:tenant_admin", Dom: "default", Obj: "tenant:*", Act: "*"},
	}

	// 定义角色继承关系
	groupings := []policyDomain.GroupingRule{
		{Sub: "role:super_admin", Role: "role:tenant_admin", Dom: "default"},
		{Sub: "role:tenant_admin", Role: "role:user", Dom: "default"},
	}

	if err := casbinPort.AddPolicy(ctx, policies...); err != nil {
		return fmt.Errorf("add policy rules: %w", err)
	}
	if err := casbinPort.AddGroupingPolicy(ctx, groupings...); err != nil {
		return fmt.Errorf("add grouping rules: %w", err)
	}

	deps.Logger.Infow("✅ Casbin 策略规则已创建",
		"policies", len(policies),
		"groupings", len(groupings),
	)
	return nil
}

// ==================== 辅助函数 ====================

// isDuplicateAssignment 检查是否是重复分配错误
func isDuplicateAssignment(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "already has role")
}
