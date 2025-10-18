// Package test 集成测试
package test

import (
	"context"
	"testing"
	"time"

	casbin "github.com/casbin/casbin/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	assignmentApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/assignment"
	policyApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/policy"
	resourceApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/resource"
	roleApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/role"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	assignmentService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/service"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driving"
	policyDriving "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driving"
	policyService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/service"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	resourceDriving "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driving"
	resourceService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/service"
	roleDriving "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driving"
	roleService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/service"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/casbin"
	assignmentInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/assignment"
	policyInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/policy"
	resourceInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/resource"
	roleInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/role"
	"github.com/fangcun-mount/iam-contracts/pkg/dominguard"
)

// TestAuthzEndToEnd 端到端集成测试
// 测试完整的授权流程：创建资源 -> 创建角色 -> 配置策略 -> 赋权 -> 权限检查
func TestAuthzEndToEnd(t *testing.T) {
	// 1. 准备测试环境
	ctx := context.Background()
	tenantID := "test-tenant-001"
	
	// 初始化数据库（使用 SQLite 内存数据库）
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	
	// 自动迁移
	err = autoMigrate(db)
	require.NoError(t, err)
	
	// 初始化 Casbin
	modelPath := createCasbinModel(t)
	casbinAdapter, err := casbin.NewCasbinAdapter(db, modelPath)
	require.NoError(t, err)
	
	// 初始化仓储
	resourceRepo := resourceInfra.NewResourceRepository(db)
	roleRepo := roleInfra.NewRoleRepository(db)
	policyVersionRepo := policyInfra.NewPolicyVersionRepository(db)
	assignmentRepo := assignmentInfra.NewAssignmentRepository(db)
	
	// 初始化领域服务
	resourceManager := resourceService.NewResourceManager(resourceRepo)
	roleManager := roleService.NewRoleManager(roleRepo)
	policyManager := policyService.NewPolicyManager(roleRepo, resourceRepo)
	assignmentManager := assignmentService.NewAssignmentManager(assignmentRepo, roleRepo)
	
	// 初始化应用服务
	resourceCommander := resourceApp.NewResourceCommandService(resourceManager, resourceRepo)
	roleCommander := roleApp.NewRoleCommandService(roleManager, roleRepo)
	policyCommander := policyApp.NewPolicyCommandService(policyManager, policyVersionRepo, casbinAdapter, nil)
	assignmentCommander := assignmentApp.NewAssignmentCommandService(assignmentManager, assignmentRepo, casbinAdapter)
	
	// 2. 创建资源
	t.Log("步骤 1: 创建资源")
	orderResource, err := resourceCommander.CreateResource(ctx, resourceDriving.CreateResourceCommand{
		Key:         "order",
		Name:        "订单",
		Description: "订单资源",
		Actions:     []string{"read", "write", "delete"},
	})
	require.NoError(t, err)
	assert.Equal(t, "order", orderResource.Key)
	t.Logf("✓ 创建资源成功: %s (ID: %d)", orderResource.Name, orderResource.ID.Uint64())
	
	// 3. 创建角色
	t.Log("步骤 2: 创建角色")
	adminRole, err := roleCommander.CreateRole(ctx, roleDriving.CreateRoleCommand{
		Name:        "order-admin",
		DisplayName: "订单管理员",
		Description: "可以管理所有订单",
		TenantID:    tenantID,
	})
	require.NoError(t, err)
	assert.Equal(t, "order-admin", adminRole.Name)
	t.Logf("✓ 创建角色成功: %s (ID: %d)", adminRole.DisplayName, adminRole.ID.Uint64())
	
	// 4. 配置策略规则
	t.Log("步骤 3: 配置策略规则")
	err = policyCommander.AddPolicyRule(ctx, policyDriving.AddPolicyRuleCommand{
		RoleID:     adminRole.ID.Uint64(),
		ResourceID: orderResource.ID,
		Action:     "read",
		TenantID:   tenantID,
		ChangedBy:  "system",
		Reason:     "初始化权限",
	})
	require.NoError(t, err)
	t.Log("✓ 添加策略规则成功: order-admin -> order:read")
	
	err = policyCommander.AddPolicyRule(ctx, policyDriving.AddPolicyRuleCommand{
		RoleID:     adminRole.ID.Uint64(),
		ResourceID: orderResource.ID,
		Action:     "write",
		TenantID:   tenantID,
		ChangedBy:  "system",
		Reason:     "初始化权限",
	})
	require.NoError(t, err)
	t.Log("✓ 添加策略规则成功: order-admin -> order:write")
	
	// 5. 给用户赋权
	t.Log("步骤 4: 给用户赋权")
	userID := "user-alice"
	assignmentResult, err := assignmentCommander.Grant(ctx, driving.GrantCommand{
		SubjectType: assignment.SubjectTypeUser,
		SubjectID:   userID,
		RoleID:      adminRole.ID.Uint64(),
		TenantID:    tenantID,
		GrantedBy:   "system",
	})
	require.NoError(t, err)
	assert.NotNil(t, assignmentResult)
	t.Logf("✓ 用户赋权成功: %s -> %s", userID, adminRole.DisplayName)
	
	// 6. 权限检查（使用 DomainGuard）
	t.Log("步骤 5: 权限检查")
	
	// 创建 DomainGuard
	enforcer := casbinAdapter.GetEnforcer()
	guard, err := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: enforcer,
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)
	
	// 检查用户是否有 order:read 权限
	allowed, err := guard.CheckPermission(ctx, userID, tenantID, "order", "read")
	require.NoError(t, err)
	assert.True(t, allowed, "用户应该有 order:read 权限")
	t.Log("✓ 权限检查通过: user-alice 有 order:read 权限")
	
	// 检查用户是否有 order:write 权限
	allowed, err = guard.CheckPermission(ctx, userID, tenantID, "order", "write")
	require.NoError(t, err)
	assert.True(t, allowed, "用户应该有 order:write 权限")
	t.Log("✓ 权限检查通过: user-alice 有 order:write 权限")
	
	// 检查用户是否有 order:delete 权限（没有授予）
	allowed, err = guard.CheckPermission(ctx, userID, tenantID, "order", "delete")
	require.NoError(t, err)
	assert.False(t, allowed, "用户不应该有 order:delete 权限")
	t.Log("✓ 权限检查通过: user-alice 没有 order:delete 权限（符合预期）")
	
	// 7. 撤销权限
	t.Log("步骤 6: 撤销权限")
	err = assignmentCommander.RevokeByID(ctx, driving.RevokeByIDCommand{
		AssignmentID: assignmentResult.ID,
		TenantID:     tenantID,
	})
	require.NoError(t, err)
	t.Log("✓ 撤销权限成功")
	
	// 8. 验证权限已撤销
	t.Log("步骤 7: 验证权限已撤销")
	allowed, err = guard.CheckPermission(ctx, userID, tenantID, "order", "read")
	require.NoError(t, err)
	assert.False(t, allowed, "权限撤销后，用户不应该有 order:read 权限")
	t.Log("✓ 权限检查通过: user-alice 已没有 order:read 权限")
	
	t.Log("\n🎉 端到端集成测试通过！")
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&resourceInfra.ResourcePO{},
		&roleInfra.RolePO{},
		&policyInfra.PolicyVersionPO{},
		&assignmentInfra.AssignmentPO{},
		&casbin.CasbinRule{},
	)
}

// createCasbinModel 创建 Casbin 模型文件
func createCasbinModel(t *testing.T) string {
	modelContent := `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
`
	tmpFile := t.TempDir() + "/model.conf"
	err := os.WriteFile(tmpFile, []byte(modelContent), 0644)
	require.NoError(t, err)
	return tmpFile
}
