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

// resourceSeed 资源种子数据
type resourceSeed struct {
	Key         string
	DisplayName string
	AppName     string
	Domain      string
	Type        string
	Actions     []string
	Description string
}

// assignmentSeed 角色分配种子数据
type assignmentSeed struct {
	SubjectAlias string
	RoleID       uint64
	TenantID     string
}

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
	resourceRepo := resourceMysql.NewResourceRepository(deps.DB)
	resourceManager := resourceService.NewResourceManager(resourceRepo)
	resourceCommander := resourceApp.NewResourceCommandService(resourceManager, resourceRepo)
	resourceQueryer := resourceApp.NewResourceQueryService(resourceRepo)

	resources := []resourceSeed{
		{
			Key:         "uc:users",
			DisplayName: "用户管理",
			AppName:     "iam",
			Domain:      "uc",
			Type:        "collection",
			Actions:     []string{"create", "read", "update", "delete", "list"},
			Description: "用户中心的用户管理权限",
		},
		{
			Key:         "uc:children",
			DisplayName: "儿童管理",
			AppName:     "iam",
			Domain:      "uc",
			Type:        "collection",
			Actions:     []string{"create", "read", "update", "delete", "list"},
			Description: "用户中心的儿童档案管理权限",
		},
		{
			Key:         "uc:guardianships",
			DisplayName: "监护关系管理",
			AppName:     "iam",
			Domain:      "uc",
			Type:        "collection",
			Actions:     []string{"create", "read", "update", "delete", "list", "revoke"},
			Description: "用户中心的监护关系管理权限",
		},
		{
			Key:         "authz:roles",
			DisplayName: "角色管理",
			AppName:     "iam",
			Domain:      "authz",
			Type:        "collection",
			Actions:     []string{"create", "read", "update", "delete", "list", "assign"},
			Description: "授权模块的角色管理权限",
		},
		{
			Key:         "authz:policies",
			DisplayName: "策略管理",
			AppName:     "iam",
			Domain:      "authz",
			Type:        "collection",
			Actions:     []string{"create", "read", "update", "delete", "list"},
			Description: "授权模块的策略管理权限",
		},
	}

	for _, seed := range resources {
		if res, err := resourceQueryer.GetResourceByKey(ctx, seed.Key); err == nil && res != nil {
			state.Resources[seed.Key] = res.ID.Uint64()
			continue
		}

		created, err := resourceCommander.CreateResource(ctx, resourceDriving.CreateResourceCommand{
			Key:         seed.Key,
			DisplayName: seed.DisplayName,
			AppName:     seed.AppName,
			Domain:      seed.Domain,
			Type:        seed.Type,
			Actions:     seed.Actions,
			Description: seed.Description,
		})
		if err != nil {
			return fmt.Errorf("create resource %s: %w", seed.Key, err)
		}
		state.Resources[seed.Key] = created.ID.Uint64()
	}

	deps.Logger.Infow("✅ 授权资源数据已创建", "count", len(resources))
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

	assignments := []assignmentSeed{
		{SubjectAlias: "admin", RoleID: 1, TenantID: "default"},
		{SubjectAlias: "zhangsan", RoleID: 3, TenantID: "default"},
		{SubjectAlias: "wangwu", RoleID: 3, TenantID: "default"},
	}

	for _, seed := range assignments {
		userID := state.Users[seed.SubjectAlias]
		if userID == "" {
			return fmt.Errorf("user alias %s not found for assignment", seed.SubjectAlias)
		}

		cmd := assignmentDriving.GrantCommand{
			SubjectType: assignmentDomain.SubjectTypeUser,
			SubjectID:   userID,
			RoleID:      seed.RoleID,
			TenantID:    seed.TenantID,
			GrantedBy:   "system",
		}

		if _, err := assignmentCommander.Grant(ctx, cmd); err != nil {
			if !isDuplicateAssignment(err) {
				return fmt.Errorf("grant role %d to %s: %w", seed.RoleID, seed.SubjectAlias, err)
			}
		}
	}

	deps.Logger.Infow("✅ 角色分配数据已创建", "count", len(assignments))
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
